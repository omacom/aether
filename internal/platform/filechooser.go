package platform

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalBusName     = "org.freedesktop.portal.Desktop"
	portalObjectPath  = "/org/freedesktop/portal/desktop"
	portalOpenFile    = "org.freedesktop.portal.FileChooser.OpenFile"
	portalRequestIfc  = "org.freedesktop.portal.Request"
	portalResponseSig = "Response"

	// portalCallTimeout bounds the OpenFile method call only. The portal
	// returns a request handle at once, so a slow reply means the portal is
	// stuck. The wait for the user's pick has no limit.
	portalCallTimeout = 10 * time.Second
)

// FileFilter is one entry in a file chooser's filter list.
type FileFilter struct {
	Name     string
	Patterns []string
}

// FileChooserOptions configures PortalOpenFile.
type FileChooserOptions struct {
	Title         string
	Directory     bool
	CurrentFolder string
	Filters       []FileFilter
}

// portalFilter matches the portal's a(sa(us)) filter signature.
type portalFilter struct {
	Name  string
	Rules []portalFilterRule
}

type portalFilterRule struct {
	Type    uint32 // 0 = glob pattern, 1 = MIME type
	Pattern string
}

var portalRequestCounter atomic.Uint64

// PortalOpenFile asks xdg-desktop-portal's FileChooser for one file or
// directory, so the user's configured picker (GTK, KDE, a terminal file
// manager, ...) is used. It returns "" with a nil error when the user cancels,
// and an error when no portal answers or the backend fails, so callers can
// fall back to a toolkit dialog.
func PortalOpenFile(ctx context.Context, opts FileChooserOptions) (string, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return "", fmt.Errorf("connect to session bus: %w", err)
	}

	token := fmt.Sprintf("aether%d", portalRequestCounter.Add(1))
	expected := portalRequestPath(conn.Names()[0], token)

	// Subscribe before calling: the portal may answer before OpenFile returns.
	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	if err := addResponseMatch(conn, expected); err != nil {
		return "", err
	}
	defer removeResponseMatch(conn, expected)

	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"modal":        dbus.MakeVariant(true),
		"directory":    dbus.MakeVariant(opts.Directory),
	}
	if filters := toPortalFilters(opts.Filters); len(filters) > 0 {
		options["filters"] = dbus.MakeVariant(filters)
	}
	if opts.CurrentFolder != "" {
		options["current_folder"] = dbus.MakeVariant(append([]byte(opts.CurrentFolder), 0))
	}

	var handle dbus.ObjectPath
	callCtx, cancel := context.WithTimeout(ctx, portalCallTimeout)
	call := conn.Object(portalBusName, portalObjectPath).CallWithContext(callCtx, portalOpenFile, 0, "", opts.Title, options)
	err = call.Store(&handle)
	cancel()
	if err != nil {
		return "", fmt.Errorf("portal OpenFile: %w", err)
	}
	// The spec lets the returned handle differ from the handle_token path.
	if handle != expected {
		if err := addResponseMatch(conn, handle); err != nil {
			return "", err
		}
		defer removeResponseMatch(conn, handle)
	}

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case sig, ok := <-signals:
			if !ok {
				return "", fmt.Errorf("session bus closed while waiting for portal")
			}
			if sig.Path != handle || sig.Name != portalRequestIfc+"."+portalResponseSig {
				continue
			}
			return parsePortalResponse(sig.Body)
		}
	}
}

func addResponseMatch(conn *dbus.Conn, path dbus.ObjectPath) error {
	if err := conn.AddMatchSignal(responseMatch(path)...); err != nil {
		return fmt.Errorf("subscribe to portal response: %w", err)
	}
	return nil
}

func removeResponseMatch(conn *dbus.Conn, path dbus.ObjectPath) {
	_ = conn.RemoveMatchSignal(responseMatch(path)...)
}

func responseMatch(path dbus.ObjectPath) []dbus.MatchOption {
	return []dbus.MatchOption{
		dbus.WithMatchSender(portalBusName),
		dbus.WithMatchObjectPath(path),
		dbus.WithMatchInterface(portalRequestIfc),
		dbus.WithMatchMember(portalResponseSig),
	}
}

// portalRequestPath builds the Request object path the portal will use for
// handle_token, per the org.freedesktop.portal.Request documentation.
func portalRequestPath(uniqueName, token string) dbus.ObjectPath {
	sender := strings.ReplaceAll(strings.TrimPrefix(uniqueName, ":"), ".", "_")
	return dbus.ObjectPath(portalObjectPath + "/request/" + sender + "/" + token)
}

// toPortalFilters converts filters whose patterns may be ";"-joined
// (Wails style, e.g. "*.jpg;*.png") into portal glob rules.
func toPortalFilters(filters []FileFilter) []portalFilter {
	out := make([]portalFilter, 0, len(filters))
	for _, f := range filters {
		pf := portalFilter{Name: f.Name}
		for _, joined := range f.Patterns {
			for _, p := range strings.Split(joined, ";") {
				if p = strings.TrimSpace(p); p != "" {
					pf.Rules = append(pf.Rules, portalFilterRule{Type: 0, Pattern: p})
				}
			}
		}
		if len(pf.Rules) > 0 {
			out = append(out, pf)
		}
	}
	return out
}

// parsePortalResponse reads the (u response, a{sv} results) body of a
// Request.Response signal. Response 1 (cancelled) yields "". Response 2
// (ended in another way) is an error, because portal backends send it when
// they fail, and the caller then falls back to the toolkit dialog.
func parsePortalResponse(body []interface{}) (string, error) {
	if len(body) != 2 {
		return "", fmt.Errorf("unexpected portal response: %v", body)
	}
	code, ok := body[0].(uint32)
	if !ok {
		return "", fmt.Errorf("unexpected portal response code: %v", body[0])
	}
	switch code {
	case 0:
	case 1:
		return "", nil
	default:
		return "", fmt.Errorf("portal ended the request with response %d", code)
	}
	results, ok := body[1].(map[string]dbus.Variant)
	if !ok {
		return "", fmt.Errorf("unexpected portal results: %v", body[1])
	}
	uris, ok := results["uris"].Value().([]string)
	if !ok || len(uris) == 0 {
		return "", nil
	}
	return fileURIToPath(uris[0])
}

// fileURIToPath decodes a file:// URI. url.Parse keeps "+" literal, which
// url.QueryUnescape would turn into a space.
func fileURIToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse portal uri %q: %w", uri, err)
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("portal returned non-file uri %q", uri)
	}
	return u.Path, nil
}
