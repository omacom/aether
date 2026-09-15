package template

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"aether/internal/platform"
)

// customAppConfig represents the config.json structure inside a custom app folder.
type customAppConfig struct {
	Template    string `json:"template"`
	Destination string `json:"destination"`
}

// ProcessCustomApps scans ~/.config/aether/custom/ for app-specific templates,
// processes each template with variable substitution, writes the output to
// themeDir, creates destination symlinks, and runs any post-apply.sh scripts.
// App failures are collected and returned; failed installs never run their hook.
//
// Directory structure expected:
//
//	~/.config/aether/custom/
//	  cava/
//	    config.json      -> {"template": "theme.ini", "destination": "~/.config/cava/themes/aether"}
//	    theme.ini        -> Template file with {color} variables
//	    post-apply.sh    -> Optional script to run after apply
func ProcessCustomApps(themeDir string, variables map[string]string) error {
	customDir := platform.CustomDir()

	entries, err := os.ReadDir(customDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var errs []error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appName := entry.Name()
		appPath := filepath.Join(customDir, appName)

		if err := processCustomApp(appPath, appName, themeDir, variables); err != nil {
			errs = append(errs, fmt.Errorf("custom app %q: %w", appName, err))
		}
	}

	return errors.Join(errs...)
}

// processCustomApp handles a single custom app directory: reads config.json,
// processes the template, writes output, creates symlink, and runs post-apply.
func processCustomApp(appPath, appName, themeDir string, variables map[string]string) error {
	// Read config.json
	configPath := filepath.Join(appPath, "config.json")
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		log.Printf("[%s] Missing config.json, skipping", appName)
		return nil
	}
	if err != nil {
		return err
	}

	var config customAppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if config.Template == "" {
		return fmt.Errorf("config.json missing 'template' field")
	}

	// Read and process the template
	templatePath := filepath.Join(appPath, config.Template)
	content, err := platform.ReadText(templatePath)
	if err != nil {
		return err
	}

	processed := ProcessTemplate(content, variables)

	// Write processed output to theme directory
	outputFileName := appName + "-" + config.Template
	outputPath, err := filepath.Abs(filepath.Join(themeDir, outputFileName))
	if err != nil {
		return err
	}
	if err := platform.WriteText(outputPath, processed); err != nil {
		return err
	}
	log.Printf("[%s] Processed template: %s", appName, config.Template)

	// Create destination symlink if configured
	if config.Destination != "" {
		destPath := expandHome(config.Destination)
		if err := platform.CreateSymlink(outputPath, destPath); err != nil {
			return fmt.Errorf("install symlink at %s: %w", destPath, err)
		}
		log.Printf("[%s] Symlinked -> %s", appName, destPath)
	}

	// Run post-apply.sh if it exists
	postApplyPath := filepath.Join(appPath, "post-apply.sh")
	if platform.FileExists(postApplyPath) {
		if err := platform.RunAsync("bash", postApplyPath); err != nil {
			return fmt.Errorf("start post-apply.sh: %w", err)
		}
		log.Printf("[%s] Executed post-apply.sh", appName)
	}

	return nil
}

// ReadCustomOverride checks if a custom override template exists for the given
// file name in ~/.config/aether/custom/. If found, returns its content and true.
// If not found, returns empty string and false.
func ReadCustomOverride(fileName string) (string, bool) {
	customPath := filepath.Join(platform.CustomDir(), fileName)
	if !platform.FileExists(customPath) {
		return "", false
	}
	content, err := platform.ReadText(customPath)
	if err != nil {
		log.Printf("Error reading custom override %s: %v", fileName, err)
		return "", false
	}
	log.Printf("Using custom override for %s", fileName)
	return content, true
}

// expandHome replaces a leading "~/" with the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
