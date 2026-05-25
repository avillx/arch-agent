package searxng

import (
	"arch-agent/internal/tools/search"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

type WebSearchEngine interface {
	Search(string, int) ([]search.Result, error)
}

type SearXSearch struct {
	client     *http.Client
	hostURL    string
	hostScheme string
}

func NewSearXSearch(hostScheme, hostURL string) *SearXSearch {
	return &SearXSearch{
		client:     http.DefaultClient,
		hostURL:    hostURL,
		hostScheme: hostScheme,
	}
}

func (s *SearXSearch) Search(ctx context.Context, query string, numResults int) ([]search.Result, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.buildURL(query), http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseResponse(data, numResults)

}

func parseResponse(raw []byte, limitResult int) ([]search.Result, error) {
	var response struct {
		Query           string `json:"query"`
		NumberOfResults int    `json:"number_of_results"`
		Results         []struct {
			Url     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"results"`
	}

	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}

	results := make([]search.Result, min(len(response.Results), limitResult))
	for i := range len(results) {
		results[i] = search.Result{
			Title:   response.Results[i].Title,
			Link:    response.Results[i].Url,
			Snippet: response.Results[i].Content,
		}
	}

	return results, nil
}

func (s *SearXSearch) buildURL(query string) string {

	u := &url.URL{
		Scheme: s.hostScheme,
		Host:   s.hostURL,
		Path:   "/search",
	}

	q := u.Query()
	q.Set("q", url.QueryEscape(query))
	q.Set("format", "json")

	u.RawQuery = q.Encode()

	return u.String()
}
