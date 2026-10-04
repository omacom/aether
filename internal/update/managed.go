package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// packageUpdateCommand is set at build time with
// -ldflags "-X aether/internal/update.packageUpdateCommand=<command>".
// It overrides the runtime package detection.
var packageUpdateCommand string

var detectedUpdateCommand = sync.OnceValue(detectPackageUpdateCommand)

// UpdateCommand returns the command that updates a package-managed Aether.
// It returns "" when Aether updates itself.
func UpdateCommand() string {
	if packageUpdateCommand != "" {
		return packageUpdateCommand
	}
	return detectedUpdateCommand()
}

// ManagedUpgradeError returns an error when a package manager owns Aether,
// because a self-upgrade would overwrite a file that the package owns.
func ManagedUpgradeError() error {
	command := UpdateCommand()
	if command == "" {
		return nil
	}
	return fmt.Errorf("aether is managed by your package manager. To update it, run: %s", command)
}

// detectPackageUpdateCommand returns an update command when pacman owns the
// running executable. Omarchy and AUR helpers also update AUR packages, so
// they come before plain pacman.
func detectPackageUpdateCommand() string {
	if runtime.GOOS != "linux" || !commandExists("pacman") {
		return ""
	}
	executable, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if exec.CommandContext(ctx, "pacman", "-Qqo", executable).Run() != nil {
		return ""
	}
	if commandExists("omarchy-update") {
		return "omarchy-update"
	}
	for _, helper := range []string{"yay", "paru"} {
		if commandExists(helper) {
			return helper + " -Syu"
		}
	}
	return "sudo pacman -Syu"
}
