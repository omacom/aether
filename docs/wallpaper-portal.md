# Set as Background

Aether can handle "Set as Background" in file managers and image viewers. It applies the picture with a generated theme, as `aether --generate` does.

These apps hand the picture to xdg-desktop-portal. On GNOME the portal sets the wallpaper itself. On Hyprland and other window managers no portal backend does, so the menu entry does nothing.

## Setup

`make install` installs two files:

```text
~/.local/share/xdg-desktop-portal/portals/aether.portal
~/.local/share/dbus-1/services/org.freedesktop.impl.portal.desktop.aether.service
```

Packages install them to `/usr/share/xdg-desktop-portal/portals/` and `/usr/share/dbus-1/services/`. D-Bus starts `aether --wallpaper-portal` when an app asks for it.

Then select Aether as the Wallpaper backend for your desktop. On Hyprland, add the last line to `~/.config/xdg-desktop-portal/hyprland-portals.conf`. Without it, the portal picks the first configured backend that offers Wallpaper (usually `gtk`, which only works on GNOME).

```ini
[preferred]
default=hyprland;gtk
org.freedesktop.impl.portal.Wallpaper=aether
```

Copy the `default=` line from `/usr/share/xdg-desktop-portal/hyprland-portals.conf`, because the user file replaces it. For other desktops use `<desktop>-portals.conf`, where `<desktop>` is `$XDG_CURRENT_DESKTOP` in lowercase.

Restart the portal:

```bash
systemctl --user restart xdg-desktop-portal
```

The first time an app sets a background, the portal asks whether apps may change it. Choose Allow.

## Limits

- Aether themes the desktop background. A request for the lock screen only is rejected.
- Only local still images are accepted.
