# Standalone Usage

Aether runs on any Linux desktop (and macOS for theme file generation). On
Linux, Apply writes theme files for the desktop **targets** below, adds the
include lines they need to your own configs, and reloads the apps. Nothing
depends on a particular distribution or configuration framework.

On macOS, theme files are generated to `~/Library/Application Support/aether/theme/`.

## Theme Output Location

The generated template files live in:

```
~/.config/aether/theme/
```

Targets additionally write into the app's own config directory (see the table).
Existing files that Aether edits are backed up first, once per apply.

## Targets

A target is detected when its app (or config) is present. Undetected targets
show as "not found" in the sidebar and are skipped. Each target has one toggle
in the GUI (Desktop targets, under App templates); turning one off is saved as
`includedApps.<id> = false` in `settings.json` and that target is neither
written nor reloaded.

| Target | Id | Needs | Writes |
| --- | --- | --- | --- |
| Yozakura shell | `yozakura` | always | `~/.local/state/yozakura/theme.json` |
| Hyprland | `hyprland` | `hyprctl` and `~/.config/hypr/hyprland.lua` | `~/.config/hypr/aether-colors.lua` (+ one `dofile` line in `hyprland.lua`) |
| awww wallpaper | `awww` | `awww` | sets the wallpaper, only when it differs |
| Kitty | `kitty` | `kitty` | `~/.config/kitty/aether-theme.conf` (+ include in `kitty.conf`) |
| GTK 3/4 | `gtk` | always | `~/.config/gtk-{3,4}.0/aether-colors.css` (+ import in `gtk.css`) |
| SwayNC | `swaync` | `swaync-client` | `~/.config/swaync/colors.css` (+ import in `style.css`) |
| Rofi | `rofi` | `rofi` | `~/.config/rofi/aether-colors.rasi` (+ import in `config.rasi`) |
| Neovim | `neovim` | `~/.config/nvim` | `~/.config/nvim/lua/aether/theme.lua` |
| btop | `btop` | `btop` | `~/.config/btop/themes/aether.theme` (+ `btop.conf` theme key) |
| VS Code | `vscode` | `code` | Aether color theme extension under `~/.vscode/extensions` |
| Qt / KDE | `qt` | `~/.config/kdeglobals` or `dolphin` | `~/.config/kdeglobals` |
| fish | `fish` | `fish` | fish color variables file |
| Starship | `starship` | `starship` | `~/.config/starship.toml` |
| Hyprlock | `hyprlock` | `hyprlock` | `~/.config/hypr/aether-hyprlock.conf` (+ source in `hyprlock.conf`) |

The same app keys (for example `kitty`, `neovim`) also control the legacy
template files in `~/.config/aether/theme/`, so there is exactly one toggle per
app.

## Yozakura themes

Import the curated Yozakura `themes.json` as blueprints (existing blueprints are
kept unless you pass `--force`; the file's default theme becomes your default
blueprint):

```bash
aether --import-yozakura ~/.config/quickshell/yozakura/themes/themes.json
```

Blueprints can carry **match wallpapers**: case-insensitive filename patterns
(set in the Save blueprint dialog, comma separated). Generating from a matching
wallpaper loads that blueprint instead of extracting a palette.

Check the readability of the Yozakura roles (text on surface, muted, accent and
so on) for one blueprint or all of them:

```bash
aether --check-contrast Icho
aether --check-contrast --all
```

## Using Aether from a shell

`--generate` extracts or resolves a theme, applies it, and with `--json` prints
the resulting theme object to stdout, which is what the Yozakura shell reads:

```bash
aether --generate ~/Wallpapers/sunset.jpg --json
```

Use `--no-vscode`, `--no-neovim` or the GUI toggles to leave a target alone.

## Manual Integration

If you disable a target, or use an app Aether has no target for, the generated
files in `~/.config/aether/theme/` can be included by hand.

### Waybar

```css
@import url("file:///home/YOUR_USER/.config/aether/theme/waybar.css");
```

### Wallpaper (swaybg)

```bash
swaybg -i ~/.config/aether/theme/backgrounds/wallpaper.jpg -m fill
```

## Custom Templates

Add custom application templates in:

```
~/.config/aether/custom/
```

See [Custom Apps](custom-apps.md) for details on creating custom templates.
