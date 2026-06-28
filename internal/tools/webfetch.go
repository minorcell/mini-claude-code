package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/minorcell/mini-claude-code/internal/provider"
)

var (
	htmlStripper  = regexp.MustCompile(`(?s)<script.*?</script>|<style.*?</style>|<[^>]+>`)
	spaceStripper = regexp.MustCompile(`\s+`)
)

type WebFetchTool struct {
	client       *http.Client
	maxBodyBytes int
}

type webFetchArgs struct {
	URL string `json:"url"`
}

func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		maxBodyBytes: 64 * 1024,
	}
}

func (t *WebFetchTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "webfetch",
		Description: "Fetch a web page or API response over HTTP(S).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "Absolute http or https URL.",
				},
			},
			"required": []string{"url"},
		},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, invocation Invocation) (Result, error) {
	var args webFetchArgs
	if err := json.Unmarshal(invocation.Arguments, &args); err != nil {
		return Result{}, fmt.Errorf("decode webfetch arguments: %w", err)
	}

	parsedURL, err := url.Parse(args.URL)
	if err != nil {
		return Result{}, fmt.Errorf("parse url %q: %w", args.URL, err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return Result{}, fmt.Errorf("unsupported url scheme %q", parsedURL.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "mini-claude-code/0.1")

	resp, err := t.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("fetch url %q: %w", args.URL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(t.maxBodyBytes+1)))
	if err != nil {
		return Result{}, fmt.Errorf("read response body: %w", err)
	}

	truncated := len(body) > t.maxBodyBytes
	if truncated {
		body = body[:t.maxBodyBytes]
	}

	contentType := resp.Header.Get("Content-Type")
	output := normalizeFetchedContent(string(body), contentType)
	metadata := map[string]any{
		"url":          parsedURL.String(),
		"status":       resp.Status,
		"content_type": contentType,
		"truncated":    truncated,
	}

	if resp.StatusCode >= 400 {
		return Result{
			Output:   output,
			Metadata: metadata,
		}, fmt.Errorf("unexpected status %s", resp.Status)
	}

	return Result{
		Output:   output,
		Metadata: metadata,
	}, nil
}

func normalizeFetchedContent(body string, contentType string) string {
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		body = htmlStripper.ReplaceAllString(body, " ")
		body = spaceStripper.ReplaceAllString(body, " ")
		body = strings.TrimSpace(body)
	}
	if body == "" {
		return "(empty response body)"
	}
	return body
}
