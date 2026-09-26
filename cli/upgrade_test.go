package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"aether/internal/update"
)

// Run with -ldflags '-X aether/internal/update.packageUpdateCommand=omarchy-update'.
func TestManagedUpgradeBeforeDevVersion(t *testing.T) {
	if update.UpdateCommand() == "" {
		t.Skip("run with the package update linker flag")
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = previous }()

	code := runUpgrade(nil)
	_ = w.Close()
	output, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 || !strings.Contains(string(output), "Updates are managed by your distribution. Run: omarchy-update") || strings.Contains(string(output), "development build") {
		t.Fatalf("code = %d, stderr = %q", code, output)
	}
}
