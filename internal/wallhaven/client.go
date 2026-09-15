package wallhaven

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"aether/internal/platform"
	"aether/internal/wallpaper"
)

const (
	baseURL                   = "https://wallhaven.cc/api/v1"
	maxAPIResponseBytes int64 = 4 << 20
	maxErrorBytes       int64 = 4 << 10
	maxThumbnailBytes   int64 = 5 << 20
)

// Client is an HTTP client for the wallhaven.cc API.
type Client struct {
	http   *http.Client
	apiKey string
}

// NewClient creates a new wallhaven API client.
func NewClient() *Client {
	client := wallpaper.NewPublicHTTPClient()
	client.Timeout = 30 * time.Second
	return &Client{http: client}
}

// SetAPIKey sets the optional API key used for authenticated requests.
func (c *Client) SetAPIKey(key string) { c.apiKey = key }

// Search searches wallhaven.cc with the given params and returns the results.
func (c *Client) Search(params SearchParams) (*SearchResult, error) {
	q := url.Values{}

	if params.Query != "" {
		q.Set("q", params.Query)
	}

	categories := params.Categories
	if categories == "" {
		categories = "111"
	}
	q.Set("categories", categories)

	purity := params.Purity
	if purity == "" {
		purity = "100"
	}
	q.Set("purity", purity)

	sorting := params.Sorting
	if sorting == "" {
		sorting = "date_added"
	}
	q.Set("sorting", sorting)

	order := params.Order
	if order == "" {
		order = "desc"
	}
	q.Set("order", order)

	page := params.Page
	if page < 1 {
		page = 1
	}
	q.Set("page", strconv.Itoa(page))

	if params.AtLeast != "" {
		q.Set("atleast", params.AtLeast)
	}
	if params.Colors != "" {
		q.Set("colors", params.Colors)
	}
	if c.apiKey != "" {
		q.Set("apikey", c.apiKey)
	}

	reqURL := baseURL + "/search?" + q.Encode()

	var result SearchResult
	if err := c.getJSON(reqURL, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SearchMultiPage fetches numPages consecutive pages concurrently and merges
// the results. Metadata is normalized so the caller sees logical pagination
// (e.g. last_page is adjusted for the multi-page fetch).
func (c *Client) SearchMultiPage(params SearchParams, numPages int) (*SearchResult, error) {
	if numPages <= 1 {
		return c.Search(params)
	}

	type pageResult struct {
		index  int
		result *SearchResult
		err    error
	}

	ch := make(chan pageResult, numPages)
	for i := 0; i < numPages; i++ {
		go func(idx int) {
			p := params
			p.Page = params.Page + idx
			res, err := c.Search(p)
			ch <- pageResult{index: idx, result: res, err: err}
		}(i)
	}

	results := make([]*SearchResult, numPages)
	for i := 0; i < numPages; i++ {
		pr := <-ch
		if pr.err != nil {
			if pr.index == 0 {
				return nil, pr.err
			}
			continue
		}
		results[pr.index] = pr.result
	}

	merged := results[0]
	if merged == nil {
		return nil, fmt.Errorf("wallhaven search failed for first page")
	}
	for i := 1; i < numPages; i++ {
		if results[i] != nil {
			merged.Data = append(merged.Data, results[i].Data...)
		}
	}

	// Normalize pagination metadata so the caller sees logical pages
	merged.Meta.CurrentPage = (params.Page-1)/numPages + 1
	merged.Meta.LastPage = (merged.Meta.LastPage + numPages - 1) / numPages

	return merged, nil
}

// wallhavenPagePattern matches wallhaven.cc page URLs and extracts the ID.
var wallhavenPagePattern = regexp.MustCompile(`^/w/([a-zA-Z0-9]+)/?$`)
var wallhavenIDPattern = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// ResolveImageURL resolves a wallhaven page URL (e.g. https://wallhaven.cc/w/j3qv85)
// to the direct image URL by querying the wallhaven API. If the URL is already a
// direct image URL it is returned as-is.
func (c *Client) ResolveImageURL(wallpaperURL string) (string, error) {
	if err := wallpaper.ValidateRemoteURL(wallpaperURL); err != nil {
		return "", err
	}
	u, _ := url.Parse(wallpaperURL)
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	matches := wallhavenPagePattern.FindStringSubmatch(u.Path)
	if (host != "wallhaven.cc" && host != "www.wallhaven.cc") || matches == nil {
		return wallpaperURL, nil
	}

	id := matches[1]
	info, err := c.Info(id)
	if err != nil {
		return "", err
	}

	if info.Path == "" {
		return "", fmt.Errorf("wallhaven API returned no image path for %s", id)
	}
	if err := wallpaper.ValidateRemoteURL(info.Path); err != nil {
		return "", fmt.Errorf("invalid wallhaven image URL: %w", err)
	}
	return info.Path, nil
}

// DownloadFromURL resolves a wallhaven URL (page or direct) and downloads the
// wallpaper. Returns the local file path.
func (c *Client) DownloadFromURL(wallpaperURL string) (string, error) {
	imageURL, err := c.ResolveImageURL(wallpaperURL)
	if err != nil {
		return "", err
	}
	return c.Download(imageURL)
}

// DownloadThumb downloads a wallhaven thumbnail URL into the dedicated cache
// dir under ~/.cache/aether/wallhaven-thumbs and returns the local path. Cached
// hits skip the HTTP round-trip.
func (c *Client) DownloadThumb(thumbURL string) (string, error) {
	destDir := filepath.Join(platform.CacheDir(), "wallhaven-thumbs")
	return c.download(thumbURL, destDir, maxThumbnailBytes)
}

// Info fetches metadata for a single wallpaper by its wallhaven ID.
func (c *Client) Info(id string) (*WallpaperInfo, error) {
	if !wallhavenIDPattern.MatchString(id) {
		return nil, fmt.Errorf("invalid wallpaper id %q", id)
	}

	reqURL := baseURL + "/w/" + id
	if c.apiKey != "" {
		reqURL += "?apikey=" + url.QueryEscape(c.apiKey)
	}

	var result struct {
		Data WallpaperInfo `json:"data"`
	}
	if err := c.getJSON(reqURL, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// Download downloads a wallpaper image to the local downloads directory.
// Returns the local file path.
func (c *Client) Download(imageURL string) (string, error) {
	return c.download(imageURL, platform.DownloadDir(), wallpaper.MaxImageBytes)
}

func (c *Client) download(rawURL, destDir string, maxBytes int64) (string, error) {
	if err := wallpaper.ValidateRemoteURL(rawURL); err != nil {
		return "", err
	}
	u, _ := url.Parse(rawURL)
	filename := path.Base(u.Path)
	if filename == "." || filename == ".." || filename == "/" ||
		strings.ContainsAny(filename, "\\\x00") {
		return "", fmt.Errorf("invalid download filename")
	}
	destPath := filepath.Join(destDir, filename)
	if err := wallpaper.DownloadFile(c.http, rawURL, destPath, maxBytes); err != nil {
		return "", err
	}
	return destPath, nil
}

func (c *Client) getJSON(reqURL string, result any) error {
	resp, err := c.http.Get(reqURL)
	if err != nil {
		return fmt.Errorf("wallhaven API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
		return fmt.Errorf("wallhaven API returned %d: %s", resp.StatusCode, body)
	}
	if resp.ContentLength > maxAPIResponseBytes {
		return fmt.Errorf("wallhaven response exceeds %d-byte limit", maxAPIResponseBytes)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAPIResponseBytes+1))
	if err != nil {
		return fmt.Errorf("failed to read wallhaven response: %w", err)
	}
	if int64(len(body)) > maxAPIResponseBytes {
		return fmt.Errorf("wallhaven response exceeds %d-byte limit", maxAPIResponseBytes)
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to decode wallhaven response: %w", err)
	}
	return nil
}
