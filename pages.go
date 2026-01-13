package confluence

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/resolute-sh/resolute/core"
	transform "github.com/resolute-sh/resolute-transform"
)

// FetchPagesInput is the input for FetchPagesActivity.
type FetchPagesInput struct {
	BaseURL  string
	Email    string
	APIToken string
	SpaceKey string
	Since    *time.Time
	Limit    int
}

// FetchPagesOutput is the output of FetchPagesActivity.
type FetchPagesOutput struct {
	Ref   core.DataRef
	Count int
}

// FetchPagesActivity fetches pages from a Confluence space and stores them.
func FetchPagesActivity(ctx context.Context, input FetchPagesInput) (FetchPagesOutput, error) {
	client := NewClient(ClientConfig{
		BaseURL:  input.BaseURL,
		Email:    input.Email,
		APIToken: input.APIToken,
	})

	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}

	pages, err := client.GetSpacePages(ctx, input.SpaceKey, limit)
	if err != nil {
		return FetchPagesOutput{}, fmt.Errorf("get space pages: %w", err)
	}

	docs := make([]transform.Document, 0, len(pages))
	for _, page := range pages {
		if input.Since != nil && page.Version.CreatedAt.Before(*input.Since) {
			continue
		}
		doc := pageToDocument(page, input.BaseURL)
		docs = append(docs, doc)
	}

	ref, err := transform.StoreDocuments(ctx, docs)
	if err != nil {
		return FetchPagesOutput{}, fmt.Errorf("store documents: %w", err)
	}

	return FetchPagesOutput{
		Ref:   ref,
		Count: len(docs),
	}, nil
}

// FetchPageInput is the input for FetchPageActivity.
type FetchPageInput struct {
	BaseURL  string
	Email    string
	APIToken string
	PageID   string
}

// FetchPageOutput is the output of FetchPageActivity.
type FetchPageOutput struct {
	Document transform.Document
	Found    bool
}

// FetchPageActivity fetches a single page by ID.
func FetchPageActivity(ctx context.Context, input FetchPageInput) (FetchPageOutput, error) {
	client := NewClient(ClientConfig{
		BaseURL:  input.BaseURL,
		Email:    input.Email,
		APIToken: input.APIToken,
	})

	page, err := client.GetPage(ctx, input.PageID)
	if err != nil {
		return FetchPageOutput{}, fmt.Errorf("get page: %w", err)
	}

	return FetchPageOutput{
		Document: pageToDocument(*page, input.BaseURL),
		Found:    true,
	}, nil
}

// SearchCQLInput is the input for SearchCQLActivity.
type SearchCQLInput struct {
	BaseURL  string
	Email    string
	APIToken string
	CQL      string
	Limit    int
}

// SearchCQLOutput is the output of SearchCQLActivity.
type SearchCQLOutput struct {
	Ref   core.DataRef
	Count int
}

// SearchCQLActivity searches for content using CQL and stores results.
func SearchCQLActivity(ctx context.Context, input SearchCQLInput) (SearchCQLOutput, error) {
	client := NewClient(ClientConfig{
		BaseURL:  input.BaseURL,
		Email:    input.Email,
		APIToken: input.APIToken,
	})

	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}

	result, err := client.SearchCQL(ctx, input.CQL, limit)
	if err != nil {
		return SearchCQLOutput{}, fmt.Errorf("search cql: %w", err)
	}

	docs := make([]transform.Document, 0, len(result.Results))
	for _, item := range result.Results {
		doc := pageToDocument(item.Content, input.BaseURL)
		docs = append(docs, doc)
	}

	ref, err := transform.StoreDocuments(ctx, docs)
	if err != nil {
		return SearchCQLOutput{}, fmt.Errorf("store documents: %w", err)
	}

	return SearchCQLOutput{
		Ref:   ref,
		Count: len(docs),
	}, nil
}

func pageToDocument(page Page, baseURL string) transform.Document {
	content := stripHTML(page.Body.Storage.Value)
	if content == "" {
		content = stripHTML(page.Body.View.Value)
	}

	pageURL := baseURL + page.Links.WebUI

	metadata := map[string]string{
		"page_id":    page.ID,
		"space_key":  page.Space.Key,
		"space_name": page.Space.Name,
		"status":     page.Status,
		"version":    fmt.Sprintf("%d", page.Version.Number),
	}

	return transform.Document{
		ID:        page.ID,
		Content:   content,
		Title:     page.Title,
		Source:    "confluence",
		URL:       pageURL,
		Metadata:  metadata,
		UpdatedAt: page.Version.CreatedAt,
	}
}

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func stripHTML(html string) string {
	text := htmlTagRegex.ReplaceAllString(html, " ")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")

	words := strings.Fields(text)
	return strings.Join(words, " ")
}

// FetchPages creates a node for fetching Confluence pages.
func FetchPages(input FetchPagesInput) *core.Node[FetchPagesInput, FetchPagesOutput] {
	return core.NewNode("confluence.FetchPages", FetchPagesActivity, input)
}

// FetchPage creates a node for fetching a single Confluence page.
func FetchPage(input FetchPageInput) *core.Node[FetchPageInput, FetchPageOutput] {
	return core.NewNode("confluence.FetchPage", FetchPageActivity, input)
}

// SearchCQL creates a node for searching Confluence with CQL.
func SearchCQL(input SearchCQLInput) *core.Node[SearchCQLInput, SearchCQLOutput] {
	return core.NewNode("confluence.SearchCQL", SearchCQLActivity, input)
}
