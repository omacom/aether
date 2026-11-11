package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"aether/internal/blueprint"
	"aether/internal/yozakura"
)

func runCheckContrast(args []string) int {
	all, args := hasFlag(args, "--all")
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	svc := blueprint.NewService()
	var bps []blueprint.Blueprint
	if name != "" && !all {
		bp, err := svc.FindByName(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		if bp == nil {
			fmt.Fprintf(os.Stderr, "Error: blueprint %q not found\n", name)
			return 1
		}
		bps = []blueprint.Blueprint{*bp}
	} else {
		var err error
		if bps, err = svc.LoadAll(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		sort.SliceStable(bps, func(i, j int) bool {
			return strings.ToLower(bps[i].Name) < strings.ToLower(bps[j].Name)
		})
	}

	bad := 0
	for _, bp := range bps {
		if len(bp.Palette.Colors) < 16 {
			fmt.Fprintf(os.Stderr, "skipping %s: no palette\n", bp.Name)
			continue
		}
		var palette [16]string
		copy(palette[:], bp.Palette.Colors)
		th := yozakura.Derive(MapColorsToRoles(palette), bp.Palette.YozakuraRoles, bp.Palette.Glass, bp.Palette.Clock)
		// Apply runs Enforce, so report what Apply would actually produce.
		for _, r := range yozakura.Enforce(&th) {
			flag := ""
			if r.State == "failed" {
				flag = fmt.Sprintf("   <-- below %.1f", r.Min)
				bad++
			}
			fmt.Printf("%-6s %-16s %5.2f  %s%s\n", bp.Name, r.Pair, r.Ratio, r.State, flag)
		}
	}
	if bad > 0 {
		return 1
	}
	return 0
}
