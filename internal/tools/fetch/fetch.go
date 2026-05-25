package fetch

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

const DefaultTimeout = 12 * time.Second

// getAgents
type FetchTool struct{}

func NewFetchTool() *FetchTool { return &FetchTool{} }

func (t *FetchTool) Name() string {
	return "fetch"
}

func (t *FetchTool) Description() string {
	return "fetch web pagens on provided URL"
}

func (t *FetchTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "url",
			Required:    true,
			Type:        agent.TypeString,
			Description: "request url",
		},
		{
			Name:        "format",
			Required:    true,
			Type:        agent.TypeString,
			Description: "format of result. if html on page is important use it. If you need only information use markdown",
			Enum:        []string{"html", "markdown"},
		},
	}
}

func (t *FetchTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {

	args, err := tools.UnwrapArgs[struct {
		URL    string `json:"url"`
		Format string `json:"format"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	// agentID := tools.MustAgentID(ctx)

	if len(args.URL) == 0 {
		return "", errors.New("at least one URL is required")
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

	var formatter func(string) string
	switch args.Format {
	case "markdown":
		formatter = htmlToMarkdown
	default:
		formatter = func(s string) string {
			return s
		}
	}

	return fetchURL(ctx, client, args.URL, formatter)

}

func fetchURL(ctx context.Context, client *http.Client, urlStr string, formatter func(string) string) (string, error) {

	if err := validateURL(urlStr); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Status: %s\n", resp.Status))
	sb.WriteString(fmt.Sprintf("Content-Type: %s\n", resp.Header.Get("Content-Type")))

	maxSize := int64(1 << 20) // 1MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	formatted := formatter(string(body))

	sb.WriteString(fmt.Sprintf("Body: %s", formatted))

	return sb.String(), nil
}

func validateURL(urlStr string) error {

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return errors.Join(fmt.Errorf("invalid URL: %v", urlStr), err)
	}

	// Check for valid URL structure
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("invalid URL: missing scheme or host %v", urlStr)
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTP and HTTPS URLs are supported. %v", urlStr)
	}

	return nil
}

func htmlToMarkdown(html string) string {
	markdown, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		return html
	}
	return markdown
}
