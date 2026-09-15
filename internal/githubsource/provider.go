package githubsource

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"aether/internal/wallpaper"
)

const githubAPIBase = "https://api.github.com"
const maxAPIBytes = 4 << 20

var repositoryPart = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,100}$`)
var thumbnailSlots = make(chan struct{}, 4)

type parsedGitHubURL struct {
	Owner  string
	Repo   string
	Branch string
	Path   string
}

// Client shares a bounded metadata cache across repository requests.
type Client struct {
	http  *http.Client
	cache *ttlCache
}

func NewClient() *Client {
	client := wallpaper.NewPublicHTTPClient()
	client.Timeout = 30 * time.Second
	return &Client{http: client, cache: newTTLCache(5*time.Minute, 32)}
}

func (c *Client) ClearCache() { c.cache.clear() }

// ListImages lists one public repository directory or image.
func (c *Client) ListImages(rawURL string) (*ListContentsResult, error) {
	location, err := parseURL(rawURL)
	if err != nil {
		return nil, err
	}
	if cached, ok := c.cache.get(rawURL); ok {
		return cached, nil
	}
	contents, err := c.listContents(location.Owner, location.Repo, location.Path, location.Branch)
	if err != nil {
		return nil, err
	}
	items := make([]ImageInfo, 0, len(contents))
	for _, entry := range contents {
		if entry.Type == "dir" {
			items = append(items, ImageInfo{Name: entry.Name, Path: entry.Path, Type: "dir"})
		} else if entry.Type == "file" && isImageFile(entry.Name) && wallpaper.ValidateRemoteURL(entry.DownloadURL) == nil {
			items = append(items, ImageInfo{Name: entry.Name, URL: entry.DownloadURL, Size: entry.Size, Type: "file", Path: entry.Path})
		}
	}
	result := &ListContentsResult{Items: items}
	c.cache.set(rawURL, result)
	return result, nil
}

func (c *Client) listContents(owner, repo, filePath, branch string) ([]githubContent, error) {
	endpoint, _ := url.Parse(githubAPIBase)
	endpoint.Path = "/repos/" + owner + "/" + repo + "/contents/" + filePath
	query := endpoint.Query()
	if branch != "" {
		query.Set("ref", branch)
	}
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Aether")
	response, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("GitHub access or rate limit reached. Try again later")
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("public repository, branch, or path not found")
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > maxAPIBytes {
		return nil, fmt.Errorf("GitHub response exceeds the size limit")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxAPIBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxAPIBytes {
		return nil, fmt.Errorf("GitHub response exceeds the size limit")
	}
	var entries []githubContent
	if err := json.Unmarshal(data, &entries); err == nil {
		return entries, nil
	}
	var entry githubContent
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("invalid GitHub response: %w", err)
	}
	return []githubContent{entry}, nil
}

func isImageFile(name string) bool { return wallpaper.IsImageFile(name) }

func parseURL(rawURL string) (*parsedGitHubURL, error) {
	if err := wallpaper.ValidateRemoteURL(rawURL); err != nil {
		return nil, err
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	host := strings.ToLower(u.Hostname())
	if strings.HasSuffix(host, ".github.io") {
		return nil, fmt.Errorf("use the GitHub repository URL instead of its Pages URL")
	}
	if host != "github.com" && host != "raw.githubusercontent.com" {
		return nil, fmt.Errorf("enter a github.com repository or raw.githubusercontent.com image URL")
	}
	if u.Port() != "" && u.Port() != "443" {
		return nil, fmt.Errorf("GitHub URLs must use the standard HTTPS port")
	}
	segments := splitPath(u.EscapedPath())
	for i, segment := range segments {
		decoded, err := url.PathUnescape(segment)
		if err != nil || decoded == "." || decoded == ".." {
			return nil, fmt.Errorf("invalid GitHub path")
		}
		segments[i] = decoded
	}
	if len(segments) < 2 {
		return nil, fmt.Errorf("GitHub URL requires an owner and repository")
	}
	result := &parsedGitHubURL{Owner: segments[0], Repo: strings.TrimSuffix(segments[1], ".git")}
	if !repositoryPart.MatchString(result.Owner) || !repositoryPart.MatchString(result.Repo) || result.Repo == "." || result.Repo == ".." {
		return nil, fmt.Errorf("invalid GitHub owner or repository")
	}
	if host == "raw.githubusercontent.com" {
		if len(segments) < 4 {
			return nil, fmt.Errorf("raw image URL requires a branch and file path")
		}
		start := 2
		if len(segments) > 5 && segments[2] == "refs" && segments[3] == "heads" {
			start = 4
		}
		result.Branch = segments[start]
		result.Path = strings.Join(segments[start+1:], "/")
	} else if len(segments) >= 3 && (segments[2] == "tree" || segments[2] == "blob") {
		if len(segments) < 4 {
			return nil, fmt.Errorf("GitHub URL requires a branch after tree or blob")
		}
		result.Branch = segments[3]
		result.Path = strings.Join(segments[4:], "/")
	} else {
		result.Path = strings.Join(segments[2:], "/")
	}
	return result, nil
}

func splitPath(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool { return r == '/' })
}

// DownloadThumbnail uses the shared public downloader and bounded image decoder.
func DownloadThumbnail(rawURL string) (*ThumbnailResult, error) {
	return downloadThumbnail(rawURL, wallpaper.DownloadToCache)
}

func downloadThumbnail(rawURL string, download func(string, int64) (string, error)) (*ThumbnailResult, error) {
	if err := wallpaper.ValidateRemoteURL(rawURL); err != nil {
		return nil, err
	}
	thumbnailSlots <- struct{}{}
	defer func() { <-thumbnailSlots }()
	path, err := download(rawURL, wallpaper.MaxImageBytes)
	if err != nil {
		return nil, err
	}
	if err := wallpaper.ValidateImageFile(path); err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	config, _, err := image.DecodeConfig(file)
	_ = file.Close()
	if err != nil {
		return nil, err
	}
	thumb, err := wallpaper.GetThumbnail(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(thumb)
	if err != nil {
		return nil, err
	}
	return &ThumbnailResult{DataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), Width: config.Width, Height: config.Height}, nil
}
