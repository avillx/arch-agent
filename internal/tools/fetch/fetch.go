package fetch

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"arch-agent/internal/types"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/PuerkitoBio/goquery"
)

const (
	DefaultTimeout     = 12 * time.Second
	PageCharsHardLimit = 32_000
)

// getAgents
type FetchTool struct{}

func NewFetchTool() *FetchTool { return &FetchTool{} }

func (t *FetchTool) Name() agent.ToolName {
	return "fetch"
}

func (t *FetchTool) Description() string {
	return "Fetch content of a web page by URL"
}

func (t *FetchTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "url",
			Required:    true,
			Type:        agent.TypeString,
			Description: "URL of the page to fetch",
		},
		{
			Name:        "format",
			Required:    false,
			Type:        agent.TypeString,
			Description: "Output format: 'rawHTML' preserves page structure, 'markdown' extracts readable content",
			Enum:        []string{"rawHTML", "markdown"},
		},
	}
}

func (t *FetchTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {

	args, err := tools.UnwrapArgs[struct {
		URL    string `json:"url"`
		Format string `json:"format,omitempty"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	// agentID := tools.MustAgentID(ctx)

	if len(args.URL) == 0 {
		return nil, errors.New("at least one URL is required")
	}

	client := &http.Client{
		Timeout:   DefaultTimeout,
		Transport: http.DefaultTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}

	var formatter func([]byte) (string, error)
	switch args.Format {
	case "rawHTML":
		formatter = func(raw []byte) (string, error) {
			return string(raw), nil
		}
	default:
		formatter = htmlToMarkdown
	}

	res, err := fetchURL(ctx, client, args.URL, formatter)
	return tools.Result(res), err

}

func fetchURL(ctx context.Context, client *http.Client, urlStr string, formatter func([]byte) (string, error)) (string, error) {

	// make request
	if err := validateURL(urlStr); err != nil {
		return "", types.NewAgentMistakeErrorf("url validation failed: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, http.NoBody)
	if err != nil {

		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("can't fetch page: status - %d", resp.StatusCode), nil
	}
	defer resp.Body.Close()

	// Unwrap result
	status := resp.Status
	contentType := resp.Header.Get("Content-Type")

	if !IsTextContentType(contentType) {
		return "", types.NewAgentMistakeErrorf("page has unsupported content-type: %s", contentType)
	}

	// Assemble Message
	var sb strings.Builder

	fmt.Fprintf(&sb, "Status: %s\n", status)
	fmt.Fprintf(&sb, "Content-Type: %s\n", contentType)

	// read body
	maxSize := int64(1 << 18) // ~250kb
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		sb.WriteString("can't read page")
		return sb.String(), fmt.Errorf("failed to read response body: %w", err)
	}

	content, err := formatter(body)
	if err != nil {
		content += fmt.Sprintf("\n\n formatting error %e", err)
	}

	// hard guardRail
	content = truncate(content, PageCharsHardLimit)

	fmt.Fprintf(&sb, "Body: %s", content)

	return sb.String(), nil
}

func validateURL(urlStr string) error {

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check for valid URL structure
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("invalid URL: missing scheme or host")
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTP and HTTPS URLs are supported")
	}

	return nil
}

func IsTextContentType(contentType string) bool {
	switch {
	case strings.HasPrefix(contentType, "text/plain"),
		strings.HasPrefix(contentType, "text/html"),
		strings.HasPrefix(contentType, "text/css"),
		strings.HasPrefix(contentType, "text/csv"),
		strings.HasPrefix(contentType, "text/xml"),
		strings.HasPrefix(contentType, "text/markdown"),
		strings.HasPrefix(contentType, "application/json"),
		strings.HasPrefix(contentType, "application/xml"),
		strings.HasPrefix(contentType, "application/xhtml+xml"),
		strings.HasPrefix(contentType, "application/javascript"),
		strings.HasPrefix(contentType, "application/x-www-form-urlencoded"):

		return true
	default:
		return false
	}
}

func htmlToMarkdown(rawHTML []byte) (string, error) {

	rawCleanHTML, err := cleanHTML(rawHTML)
	if err != nil {
		return "", err
	}

	cleanHTMLStr := string(rawCleanHTML)

	markdown, err := htmltomarkdown.ConvertString(cleanHTMLStr)
	if err != nil {
		return cleanHTMLStr, nil
	}
	return markdown, nil
}

func cleanHTML(body []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	doc.Find("script, style, nav, footer, header, aside, iframe, noscript, svg, form").Remove()

	html, err := doc.Html()
	if err != nil {
		return "", err
	}

	return html, nil
}

func truncate(s string, maxChars int) string {
	if len(s) <= maxChars {
		return s
	}

	cut := s[:maxChars]
	lastNewline := strings.LastIndex(cut, "\n")
	if lastNewline > 0 {
		return cut[:lastNewline]
	}

	return cut
}
