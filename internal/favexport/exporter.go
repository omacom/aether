// Package favexport bundles favorited wallpapers into a .zip archive.
//
// Favorites are not necessarily files: wallhaven entries store a remote URL as
// their path, so exporting has to fetch anything that is not on disk yet before
// it can archive it. That makes the operation slow enough to need progress
// reporting and cancellation, so it runs in a goroutine and emits Wails events
// the same way internal/batch does.
package favexport

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aether/internal/platform"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"
)

// maxConcurrentDownloads bounds the fetch phase. Downloads dominate the wall
// clock, but hammering wallhaven with one request per favorite is rude.
const maxConcurrentDownloads = 4

// manifestName is the metadata file written alongside the images so an archive
// records where each wallpaper came from.
const manifestName = "favorites.json"

// Item is one wallpaper to export.
type Item struct {
	Path string                 // local file path or http(s) URL
	Name string                 // preferred base name in the zip ("" derives from Path)
	Meta map[string]interface{} // type/id/resolution, copied into the manifest
}

// Skip records a favorite that could not be included.
type Skip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Result is the payload of the favorites-export-completed event.
type Result struct {
	ZipPath  string `json:"zipPath"`
	Total    int    `json:"total"`
	Exported int    `json:"exported"`
	Skipped  []Skip `json:"skipped"`
}

// Downloader fetches a remote wallpaper and returns its local path.
// *wallhaven.Client satisfies this.
type Downloader interface {
	DownloadContext(ctx context.Context, url string) (string, error)
}

// Exporter runs at most one export at a time.
type Exporter struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
	dl      Downloader
}

// New creates an exporter that resolves remote favorites through dl.
func New(dl Downloader) *Exporter {
	return &Exporter{dl: dl}
}

// Start validates the request and starts an export job.
// It returns the destination path. Events report progress and completion.
func (e *Exporter) Start(appCtx context.Context, items []Item, destDir string) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no favorites to export")
	}
	if err := platform.EnsureDir(destDir); err != nil {
		return "", fmt.Errorf("prepare export directory: %w", err)
	}

	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return "", fmt.Errorf("an export is already running")
	}

	zipPath, err := uniquePath(destDir, archiveBaseName(), ".zip")
	if err != nil {
		e.mu.Unlock()
		return "", err
	}

	// Derive from appCtx so app shutdown cancels an in-flight export.
	ctx, cancel := context.WithCancel(appCtx)
	e.cancel = cancel
	e.running = true
	e.mu.Unlock()

	go func() {
		defer func() {
			e.mu.Lock()
			e.running = false
			e.cancel = nil
			e.mu.Unlock()
			cancel()
		}()
		e.run(appCtx, ctx, items, zipPath)
	}()

	return zipPath, nil
}

// Cancel stops a running export. It is a no-op when nothing is running.
func (e *Exporter) Cancel() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel != nil {
		e.cancel()
	}
}

// IsRunning reports whether an export is in flight.
func (e *Exporter) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// resolved pairs an item with the local file backing it, or the reason it has none.
type resolved struct {
	item  Item
	local string
	skip  string
}

// run performs the export. emitCtx is the app context used for events (it must
// stay alive after cancellation so the cancelled event still reaches the UI);
// ctx is the cancellable one that governs the work itself.
func (e *Exporter) run(emitCtx, ctx context.Context, items []Item, zipPath string) {
	partPath := zipPath + ".part"

	results, cancelled := e.resolveAll(emitCtx, ctx, items)
	if cancelled {
		emit(emitCtx, "favorites-export-cancelled", nil)
		return
	}

	result, err := writeArchive(emitCtx, ctx, results, partPath, zipPath)
	switch {
	case errors.Is(err, context.Canceled):
		emit(emitCtx, "favorites-export-cancelled", nil)
	case err != nil:
		emit(emitCtx, "favorites-export-failed", map[string]interface{}{"error": err.Error()})
	default:
		emit(emitCtx, "favorites-export-completed", result)
	}
}

// resolveAll turns every item into a local file path, downloading remote ones
// with bounded concurrency. Results keep the input order.
func (e *Exporter) resolveAll(emitCtx, ctx context.Context, items []Item) ([]resolved, bool) {
	results := make([]resolved, len(items))
	sem := make(chan struct{}, maxConcurrentDownloads)

	var (
		wg   sync.WaitGroup
		done int
		mu   sync.Mutex
	)

	for i, item := range items {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil, true
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(i int, item Item) {
			defer wg.Done()
			defer func() { <-sem }()

			results[i] = e.resolve(ctx, item)

			mu.Lock()
			done++
			progress := done
			mu.Unlock()

			emitProgress(emitCtx, "download", progress, len(items), entryLabel(item))
		}(i, item)
	}

	wg.Wait()

	if ctx.Err() != nil {
		return nil, true
	}
	return results, false
}

// resolve maps a single item to a local file, fetching it when remote.
func (e *Exporter) resolve(ctx context.Context, item Item) resolved {
	if isRemote(item.Path) {
		if e.dl == nil {
			return resolved{item: item, skip: "no downloader available"}
		}
		local, err := e.dl.DownloadContext(ctx, item.Path)
		if err != nil {
			if ctx.Err() != nil {
				return resolved{item: item, skip: "cancelled"}
			}
			return resolved{item: item, skip: "download failed: " + err.Error()}
		}
		return resolved{item: item, local: local}
	}

	info, err := os.Stat(item.Path)
	if err != nil {
		return resolved{item: item, skip: "file not found"}
	}
	if !info.Mode().IsRegular() {
		return resolved{item: item, skip: "source is not a regular file"}
	}
	return resolved{item: item, local: item.Path}
}

// manifestEntry describes one archived wallpaper.
type manifestEntry struct {
	File   string                 `json:"file"`
	Source string                 `json:"source"`
	Meta   map[string]interface{} `json:"meta,omitempty"`
}

// writeArchive streams the resolved files into a zip, writing to partPath and
// renaming to zipPath only once the archive is complete.
func writeArchive(emitCtx, ctx context.Context, results []resolved, partPath, zipPath string) (Result, error) {
	out, err := os.OpenFile(partPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return Result{}, fmt.Errorf("create archive: %w", err)
	}
	// Closed explicitly below; the deferred close covers the error paths and a
	// second Close on an already-closed file is harmless here.
	defer func() {
		_ = out.Close()
		_ = os.Remove(partPath)
	}()

	zw := zip.NewWriter(out)
	defer zw.Close()
	used := map[string]bool{manifestName: true}
	manifest := make([]manifestEntry, 0, len(results))
	result := Result{ZipPath: zipPath, Total: len(results)}

	for i, r := range results {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}

		if r.skip != "" {
			result.Skipped = append(result.Skipped, Skip{Path: r.item.Path, Reason: r.skip})
			continue
		}

		name := uniqueEntryName(entryLabel(r.item), used)
		emitProgress(emitCtx, "archive", i+1, len(results), name)

		if err := addFile(ctx, zw, name, r.local); err != nil {
			return Result{}, fmt.Errorf("archive %s: %w", name, err)
		}

		result.Exported++
		manifest = append(manifest, manifestEntry{
			File:   name,
			Source: r.item.Path,
			Meta:   r.item.Meta,
		})
	}

	if result.Exported == 0 {
		_ = zw.Close()
		return Result{}, fmt.Errorf("no favorites could be exported")
	}

	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := addManifest(zw, manifest); err != nil {
		_ = zw.Close()
		return Result{}, err
	}
	if err := zw.Close(); err != nil {
		return Result{}, fmt.Errorf("finalize archive: %w", err)
	}
	if err := out.Close(); err != nil {
		return Result{}, fmt.Errorf("finalize archive: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if _, err := os.Lstat(zipPath); err == nil {
		return Result{}, fmt.Errorf("export destination already exists: %s", zipPath)
	} else if !os.IsNotExist(err) {
		return Result{}, fmt.Errorf("inspect export destination: %w", err)
	}
	if err := os.Rename(partPath, zipPath); err != nil {
		return Result{}, fmt.Errorf("finalize archive: %w", err)
	}

	return result, nil
}

// addFile copies one wallpaper into the archive. Images are stored, not
// deflated — they are already compressed, so deflate only burns CPU.
func addFile(ctx context.Context, zw *zip.Writer, name, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	defer src.Close()

	header := &zip.FileHeader{Name: name, Method: zip.Store}
	// Without an explicit mtime every entry reports 1980-01-01, which archive
	// tools surface as a corrupt-looking date.
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("inspect source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file")
	}
	header.Modified = info.ModTime()

	w, err := zw.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create archive entry: %w", err)
	}
	if _, err := io.Copy(w, &cancelableReader{ctx: ctx, source: src}); err != nil {
		return fmt.Errorf("copy source: %w", err)
	}
	return nil
}

type cancelableReader struct {
	ctx    context.Context
	source io.Reader
}

func (r *cancelableReader) Read(buf []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.source.Read(buf)
}

// addManifest writes the metadata sidecar. Unlike the images, JSON compresses.
func addManifest(zw *zip.Writer, entries []manifestEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("build manifest: %w", err)
	}
	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:     manifestName,
		Method:   zip.Deflate,
		Modified: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("build manifest: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("build manifest: %w", err)
	}
	return nil
}

// --- helpers ---

func isRemote(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// entryLabel is the preferred file name for an item inside the archive.
func entryLabel(item Item) string {
	source := item.Path
	if isRemote(source) {
		if parsed, err := url.Parse(source); err == nil {
			source = parsed.Path
		}
	}
	name := sanitizeName(item.Name)
	if name == "" {
		name = sanitizeName(filepath.Base(source))
	}
	if name == "" {
		name = "wallpaper"
	}
	// A wallhaven id is used as the display name in the UI and carries no
	// extension; borrow the one from the URL so the file stays openable.
	// sanitizeName trims dots, so the separator is re-added by hand.
	if filepath.Ext(name) == "" {
		if ext := sanitizeName(filepath.Ext(source)); ext != "" {
			name += "." + ext
		}
	}
	return name
}

// sanitizeName strips path separators and anything else that would let a
// favorite path escape the archive root or produce an unusable file name.
func sanitizeName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', 0:
			return -1
		}
		if r < 32 {
			return -1
		}
		return r
	}, name)
	return strings.Trim(strings.TrimSpace(name), ".")
}

// uniqueEntryName suffixes duplicates so two favorites with the same base name
// do not overwrite each other inside the archive.
func uniqueEntryName(name string, used map[string]bool) string {
	if !used[name] {
		used[name] = true
		return name
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", stem, i, ext)
		if !used[candidate] {
			used[candidate] = true
			return candidate
		}
	}
}

// archiveBaseName is the date-stamped stem for a favorites archive.
func archiveBaseName() string {
	return "aether-favorites-" + time.Now().Format("2006-01-02")
}

// uniquePath selects an unused name. Archive publication checks the destination again.
func uniquePath(dir, base, ext string) (string, error) {
	for i := 1; i < 1000; i++ {
		name := base + ext
		if i > 1 {
			name = fmt.Sprintf("%s-%d%s", base, i, ext)
		}
		path := filepath.Join(dir, name)
		if !platform.FileExists(path) && !platform.FileExists(path+".part") {
			return path, nil
		}
	}
	return "", fmt.Errorf("could not find an unused file name in %s", dir)
}

// emit publishes a Wails event, tolerating a context that carries no event
// manager. Wails' getEvents log.Fatalf's in that case, which would take the
// process down when the exporter runs outside the GUI (tests, CLI).
func emit(ctx context.Context, event string, payload interface{}) {
	if ctx == nil || ctx.Value("events") == nil {
		return
	}
	if payload == nil {
		wailsrt.EventsEmit(ctx, event)
		return
	}
	wailsrt.EventsEmit(ctx, event, payload)
}

func emitProgress(ctx context.Context, phase string, index, total int, name string) {
	emit(ctx, "favorites-export-progress", map[string]interface{}{
		"phase": phase,
		"index": index,
		"total": total,
		"name":  name,
	})
}
