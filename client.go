package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is a Confluence REST API client.
type Client struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
}

// ClientConfig contains configuration for creating a Confluence client.
type ClientConfig struct {
	BaseURL  string
	Email    string
	APIToken string
	Timeout  time.Duration
}

// NewClient creates a new Confluence client.
func NewClient(cfg ClientConfig) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL:  cfg.BaseURL,
		email:    cfg.Email,
		apiToken: cfg.APIToken,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Page represents a Confluence page.
type Page struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Status  string    `json:"status"`
	Title   string    `json:"title"`
	Space   Space     `json:"space"`
	Body    Body      `json:"body"`
	Version Version   `json:"version"`
	Links   PageLinks `json:"_links"`
}

// Space represents a Confluence space.
type Space struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Body represents page content.
type Body struct {
	Storage StorageBody `json:"storage"`
	View    ViewBody    `json:"view"`
}

// StorageBody is the storage format content.
type StorageBody struct {
	Value string `json:"value"`
}

// ViewBody is the view format content.
type ViewBody struct {
	Value string `json:"value"`
}

// Version represents page version.
type Version struct {
	Number    int       `json:"number"`
	When      string    `json:"when"`
	CreatedAt time.Time `json:"createdAt"`
}

// PageLinks contains page links.
type PageLinks struct {
	WebUI string `json:"webui"`
	Self  string `json:"self"`
}

// SearchResult represents a CQL search result.
type SearchResult struct {
	Results []SearchResultItem `json:"results"`
	Start   int                `json:"start"`
	Limit   int                `json:"limit"`
	Size    int                `json:"size"`
}

// SearchResultItem represents a single search result.
type SearchResultItem struct {
	Content   Page   `json:"content"`
	Title     string `json:"title"`
	Excerpt   string `json:"excerpt"`
	URL       string `json:"url"`
	ResultType string `json:"resultGlobalContainer"`
}

// SearchCQL searches for content using CQL.
func (c *Client) SearchCQL(ctx context.Context, cql string, limit int) (*SearchResult, error) {
	if limit <= 0 {
		limit = 25
	}

	endpoint := fmt.Sprintf("%s/wiki/rest/api/search?cql=%s&limit=%d&expand=content.body.storage,content.space,content.version",
		c.baseURL, url.QueryEscape(cql), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("confluence API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GetPage fetches a single page by ID.
func (c *Client) GetPage(ctx context.Context, pageID string) (*Page, error) {
	endpoint := fmt.Sprintf("%s/wiki/rest/api/content/%s?expand=body.storage,space,version",
		c.baseURL, pageID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("confluence API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var page Page
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &page, nil
}

// GetSpacePages fetches all pages in a space.
func (c *Client) GetSpacePages(ctx context.Context, spaceKey string, limit int) ([]Page, error) {
	if limit <= 0 {
		limit = 25
	}

	endpoint := fmt.Sprintf("%s/wiki/rest/api/content?spaceKey=%s&limit=%d&expand=body.storage,space,version",
		c.baseURL, spaceKey, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("confluence API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Results []Page `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Results, nil
}

func (c *Client) setAuth(req *http.Request) {
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
}
