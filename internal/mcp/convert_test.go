package mcp

import (
	"errors"
	"reflect"
	"testing"

	"arch-agent/internal/agent"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToResult(t *testing.T) {
	textResult := &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "hello"}},
	}

	multiResult := &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: "first"},
			&mcpsdk.TextContent{Text: "second"},
		},
	}

	imageResult := &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.ImageContent{Data: []byte("png-data"), MIMEType: string(agent.Png)},
		},
	}

	badImageResult := &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.ImageContent{Data: []byte("x"), MIMEType: "application/octet-stream"},
		},
	}

	wantErr := errors.New("tool failed")
	errResult := &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "boom"}},
	}
	errResult.SetError(wantErr)

	tests := []struct {
		name    string
		content *mcpsdk.CallToolResult
		want    []agent.ContentPart
		wantErr error
	}{
		{
			name:    "empty",
			content: &mcpsdk.CallToolResult{},
			want:    []agent.ContentPart{},
		},
		{
			name:    "text",
			content: textResult,
			want:    []agent.ContentPart{{Text: "hello"}},
		},
		{
			name:    "multiple text",
			content: multiResult,
			want:    []agent.ContentPart{{Text: "first"}, {Text: "second"}},
		},
		{
			name:    "image with error",
			content: errResult,
			want:    []agent.ContentPart{{Text: "boom"}},
			wantErr: wantErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toResult(tt.content)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("toResult() error = %v, want %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("toResult() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("image", func(t *testing.T) {
		got, err := toResult(imageResult)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(got))
		}
		if got[0].Text != "" {
			t.Fatalf("expected empty text, got %q", got[0].Text)
		}
		if got[0].ImageURL == "" {
			t.Fatal("expected non-empty image url")
		}
	})

	t.Run("unsupported image", func(t *testing.T) {
		got, err := toResult(badImageResult)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(got))
		}
		if got[0].ImageURL != "" {
			t.Fatalf("expected empty image url, got %q", got[0].ImageURL)
		}
		if got[0].Text == "" {
			t.Fatal("expected non-empty text describing unsupported image")
		}
	})
}
