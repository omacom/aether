package cli

import (
	"fmt"
	"os"

	"aether/internal/blueprint"
)

func runImportYozakura(args []string) int {
	force, args := hasFlag(args, "--force")
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: aether --import-yozakura <themes.json> [--force]")
		return 1
	}
	res, err := blueprint.ImportYozakuraFile(expandHome(args[0]), force)
	if res != nil {
		for _, name := range res.Imported {
			fmt.Printf("Imported blueprint: %s\n", name)
		}
		if res.Default != "" {
			fmt.Printf("Default blueprint: %s\n", res.Default)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
