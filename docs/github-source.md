# GitHub Source

Use GitHub as a wallpaper source for Aether themes.
The browser supports public repositories and the same still-image formats as Local.

## Browse a repository

1. Open the `GitHub` tab.
2. Enter a repository URL.
3. Click `Fetch`.
4. Open a directory or select `Use` on an image.

Use the name filter to narrow the current results.
The heart saves an image in Favorites.
The additional-image control adds it to the current theme's image set.
On Omarchy, `Wallpaper only` changes the background and preserves the active theme.

The command palette also provides `Go to GitHub`.
Saved repositories remain available between sessions.

## URL formats

| Source | Example |
| --- | --- |
| Repository | `https://github.com/owner/repo` |
| Branch | `https://github.com/owner/repo/tree/main` |
| Directory | `https://github.com/owner/repo/tree/main/wallpapers` |
| Image | `https://github.com/owner/repo/blob/main/image.png` |
| Raw image | `https://raw.githubusercontent.com/owner/repo/main/image.png` |

Use the source repository URL for a GitHub Pages site.
Published Pages paths can differ from repository paths.
Encode a slash inside a branch name as `%2F`.

## Downloads and previews

Aether loads thumbnails when their cards approach the visible area.
Previews use the shared download and image limits.
The metadata cache retains up to 32 responses for five minutes.

GitHub limits anonymous API requests.
If the limit is reached, wait before you retry.
The browser lists one directory at a time.
