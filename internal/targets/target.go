// Package targets writes generated theme files into the user's home, ensures
// include lines in user-owned configs, and reloads the affected apps.
package targets

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aether/internal/yozakura"
)

// Runner runs an external command and returns its output (stdout and stderr
// combined; used as the failure message when err != nil).
type Runner func(name string, args ...string) (output string, err error)

type Context struct {
	Name      string // theme display name
	ID        string // blueprint slug or "auto"
	Wallpaper string
	Vars      map[string]string // template.BuildVariables output
	Yoz       yozakura.Theme    // after Enforce
	Home      string
	Run       Runner
	Have      func(bin string) bool // PATH lookup; nil uses exec.LookPath (tests inject)
	Backups   *Backups              // one per apply run
	Templates fs.FS                 // embedded templates root (the "templates" dir)
	// Render renders an embedded template for an app through the override path
	// (custom template, per-app colours); nil falls back to plain substitution.
	Render func(templateFile, app string) (string, error)
}

// File is a generated file; Path is relative to Context.Home.
type File struct {
	Path   string
	Render func(Context) (string, error)
}

// Include: Line may hold several "\n"-separated lines; each is checked and
// inserted independently, in order. Append=false prepends missing lines at the
// top; Append=true appends them at the end. Replaces: old lines swapped in
// place for Line's first line.
type Include struct {
	Path     string
	Line     string
	Append   bool
	Replaces []string
}

type Target struct {
	ID, Label    string
	Detect       func(Context) bool
	Files        []File
	Includes     []Include
	Reload       func(Context) []string // argv; nil means no reload
	AlwaysReload bool
	// ReloadNeeded gates an AlwaysReload target: when set and it returns
	// false, the reload is skipped and the target stays "unchanged".
	ReloadNeeded func(Context) bool
	Cleanup      []string // old generated files to delete after success (relative to Home)
}

// Status.State: "written"|"unchanged"|"reloaded"|"failed"|"skipped"|"not-found"
type Status struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	State string `json:"state"`
	Path  string `json:"path"`
	Error string `json:"error,omitempty"`
}

// Apply runs every enabled, detected target and reports per-target status.
// A failing target never prevents the following ones from running.
func Apply(ctx Context, ts []Target, enabled func(id string) bool) []Status {
	if ctx.Backups == nil {
		ctx.Backups = NewBackups(ctx.Home, time.Now())
	}
	out := make([]Status, 0, len(ts))
	for _, t := range ts {
		out = append(out, applyOne(ctx, t, enabled))
	}
	return out
}

func applyOne(ctx Context, t Target, enabled func(string) bool) Status {
	st := Status{ID: t.ID, Label: t.Label}
	if len(t.Files) > 0 {
		st.Path = filepath.Join(ctx.Home, t.Files[0].Path)
	}
	if enabled != nil && !enabled(t.ID) {
		st.State = "skipped"
		return st
	}
	if t.Detect != nil && !t.Detect(ctx) {
		st.State = "not-found"
		return st
	}
	fail := func(msg string) Status {
		st.State, st.Error = "failed", msg
		return st
	}

	changed := false
	for _, f := range t.Files {
		content, err := f.Render(ctx)
		if err != nil {
			return fail(err.Error())
		}
		ch, err := WriteIfChanged(filepath.Join(ctx.Home, f.Path), content)
		if err != nil {
			return fail(err.Error())
		}
		changed = changed || ch
	}
	for _, inc := range t.Includes {
		ch, err := EnsureInclude(ctx, inc)
		if err != nil {
			return fail(err.Error())
		}
		changed = changed || ch
	}

	st.State = "unchanged"
	if changed {
		st.State = "written"
	}
	reload := changed || t.AlwaysReload
	if !changed && t.AlwaysReload && t.ReloadNeeded != nil && ctx.Run != nil && !t.ReloadNeeded(ctx) {
		reload = false
	}
	if reload && t.Reload != nil && ctx.Run != nil {
		if argv := t.Reload(ctx); len(argv) > 0 {
			stderr, err := ctx.Run(argv[0], argv[1:]...)
			if err != nil {
				if s := strings.TrimSpace(stderr); s != "" {
					return fail(s)
				}
				return fail(err.Error())
			}
			st.State = "reloaded"
		}
	}
	for _, c := range t.Cleanup {
		if err := os.Remove(filepath.Join(ctx.Home, c)); err != nil && !os.IsNotExist(err) {
			return fail(err.Error())
		}
	}
	return st
}
