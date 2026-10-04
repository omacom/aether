package cli

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"

	"github.com/godbus/dbus/v5"

	"aether/internal/theme"
)

const (
	wallpaperPortalBusName   = "org.freedesktop.impl.portal.desktop.aether"
	wallpaperPortalPath      = "/org/freedesktop/portal/desktop"
	wallpaperPortalInterface = "org.freedesktop.impl.portal.Wallpaper"

	// Portal response codes.
	portalResponseSuccess = 0
	portalResponseOther   = 2
)

// runWallpaperPortal serves the xdg-desktop-portal Wallpaper backend on the
// session bus. File managers and image viewers call it for "Set as
// Background"; Aether then applies the picture with a generated theme, as
// --generate does. D-Bus starts it on demand via the service file in
// contrib/portal.
func runWallpaperPortal(args []string) int {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: session bus: %v\n", err)
		return 1
	}
	defer conn.Close()

	if err := conn.Export(wallpaperPortal{}, wallpaperPortalPath, wallpaperPortalInterface); err != nil {
		fmt.Fprintf(os.Stderr, "Error: export portal: %v\n", err)
		return 1
	}
	reply, err := conn.RequestName(wallpaperPortalBusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: request bus name: %v\n", err)
		return 1
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		fmt.Fprintln(os.Stderr, "Wallpaper portal is already running")
		return 0
	}

	select {}
}

type wallpaperPortal struct{}

// SetWallpaperURI implements org.freedesktop.impl.portal.Wallpaper. Aether
// themes the desktop background only, so a lock-screen-only request fails.
func (wallpaperPortal) SetWallpaperURI(handle dbus.ObjectPath, appID, parentWindow, uri string, options map[string]dbus.Variant) (uint32, *dbus.Error) {
	if setOn, ok := options["set-on"].Value().(string); ok && setOn == "lockscreen" {
		return portalResponseOther, nil
	}
	path, err := wallpaperPathFromURI(uri)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return portalResponseOther, nil
	}

	self, err := os.Executable()
	if err != nil {
		self = "aether"
	}
	cmd := exec.Command(self, "--generate", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: start aether --generate: %v\n", err)
		return portalResponseOther, nil
	}
	// Reply right away: the caller only waits for the request to be
	// accepted, and generating a theme can take a few seconds.
	go cmd.Wait()
	return portalResponseSuccess, nil
}

// wallpaperPathFromURI turns the portal's file:// URI into a local image path.
func wallpaperPathFromURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid wallpaper URI %q: %w", uri, err)
	}
	if u.Scheme != "file" || u.Path == "" {
		return "", fmt.Errorf("unsupported wallpaper URI %q (expected file://)", uri)
	}
	if !theme.IsImageFile(u.Path) {
		return "", fmt.Errorf("not an image file: %s", u.Path)
	}
	return u.Path, nil
}
