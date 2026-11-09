# Troubleshooting

## App Won't Start

```bash
# Ensure webkit2gtk is installed
sudo pacman -S webkit2gtk

# Run from terminal to see errors
./aether
```

## Wallhaven Not Loading

- Check internet connection
- Rate limit: 45 req/min without API key
- Add API key in Settings for higher limits
- Clear cache: `rm -rf ~/.cache/aether/wallhaven-*`

## Theme Not Applying

Check `~/.config/aether/theme/` and the integration steps in the [standalone guide](standalone.md).

## Colors Look Wrong

- Try different extraction modes: `aether --generate wallpaper.jpg --extract-mode material`
- Check if image is too dark/bright for good extraction
- Use color adjustments in the sidebar
