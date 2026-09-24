package hooks_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/hooks"
	fstools "arch-agent/internal/tools/fs"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type pathArgs struct {
	Path string `json:"path"`
}

type moveArgs struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

var (
	readName  = (&fstools.ReadTool{}).Name()
	editName  = (&fstools.EditTool{}).Name()
	writeName = (&fstools.WriteTool{}).Name()
	moveName  = (&fstools.MoveTool{}).Name()
)

func newHook(t *testing.T, rules ...hooks.Rule) *hooks.FileAccessHook {
	t.Helper()

	h, err := hooks.NewFileAccessHook(rules...)
	if err != nil {
		t.Fatalf("NewFileAccessHook() error = %v", err)
	}
	return h
}

func pathCall(t *testing.T, name agent.ToolName, p string) *agent.ToolCall {
	t.Helper()

	raw, err := json.Marshal(pathArgs{Path: p})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return agent.NewToolCall("call-1", name, raw)
}

func moveCall(t *testing.T, src, dst string) *agent.ToolCall {
	t.Helper()

	raw, err := json.Marshal(moveArgs{Src: src, Dst: dst})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return agent.NewToolCall("call-1", moveName, raw)
}

func assertDenied(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected access denied")
	}
}

func assertAllowed(t *testing.T, tc *agent.ToolCall, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc == nil {
		t.Fatal("expected tool call returned")
	}
}

func TestNewFileAccessHook_RejectsInvalidPattern(t *testing.T) {
	_, err := hooks.NewFileAccessHook(hooks.Rule{Pattern: "[", Access: hooks.Write})
	if err == nil {
		t.Fatal("expected error for invalid pattern")
	}
}

func TestApply_IgnoresNonFileTools(t *testing.T) {
	h := newHook(t)
	tc := agent.NewToolCall("call-1", "shell", agent.ToolArguments(`{"command":"ls"}`))

	got, err := h.Apply(context.Background(), tc)

	assertAllowed(t, got, err)
	if got != tc {
		t.Fatal("expected original tool call returned")
	}
}

func TestApply_RejectsMalformedArguments(t *testing.T) {
	h := newHook(t, hooks.Rule{Pattern: "**", Access: hooks.Write})
	tc := agent.NewToolCall("call-1", readName, agent.ToolArguments(`{"path":`))

	_, err := h.Apply(context.Background(), tc)

	assertDenied(t, err)
}

func TestApply_ReadDenied(t *testing.T) {
	p := filepath.Join("shared", "file.md")
	h := newHook(t, hooks.Rule{Pattern: filepath.Join("other", "**"), Access: hooks.Read})

	_, err := h.Apply(context.Background(), pathCall(t, readName, p))

	assertDenied(t, err)
}

func TestApply_ReadAllowed(t *testing.T) {
	h := newHook(t, hooks.Rule{Pattern: filepath.Join("shared", "**"), Access: hooks.Read})
	tc := pathCall(t, readName, filepath.Join("shared", "dir", "file.md"))

	got, err := h.Apply(context.Background(), tc)

	assertAllowed(t, got, err)
	if got != tc {
		t.Fatal("expected original tool call returned")
	}
}

func TestApply_ReadAllowedWithWriteAccess(t *testing.T) {
	h := newHook(t, hooks.Rule{Pattern: filepath.Join("shared", "**"), Access: hooks.Write})
	tc := pathCall(t, readName, filepath.Join("shared", "file.md"))

	got, err := h.Apply(context.Background(), tc)

	assertAllowed(t, got, err)
}

func TestApply_WriteDenied_ReadOnlyAccess(t *testing.T) {
	p := filepath.Join("shared", "file.md")
	h := newHook(t, hooks.Rule{Pattern: filepath.Join("shared", "**"), Access: hooks.Read})

	_, err := h.Apply(context.Background(), pathCall(t, writeName, p))

	assertDenied(t, err)
}

func TestApply_WriteDenied_NoAccess(t *testing.T) {
	p := filepath.Join("shared", "file.md")
	h := newHook(t, hooks.Rule{Pattern: filepath.Join("other", "**"), Access: hooks.Write})

	_, err := h.Apply(context.Background(), pathCall(t, writeName, p))

	assertDenied(t, err)
}

func TestApply_WriteToolsAllowed(t *testing.T) {
	for _, name := range []agent.ToolName{editName, writeName} {
		h := newHook(t, hooks.Rule{Pattern: filepath.Join("shared", "**"), Access: hooks.Write})
		tc := pathCall(t, name, filepath.Join("shared", "file.md"))

		got, err := h.Apply(context.Background(), tc)

		assertAllowed(t, got, err)
		if got != tc {
			t.Fatalf("%s: expected original tool call returned", name)
		}
	}
}

func TestApply_ReadOnlyBlocksWriteTools(t *testing.T) {
	p := filepath.Join("shared", "file.md")
	h := newHook(t, hooks.Rule{Pattern: filepath.Join("shared", "**"), Access: hooks.Read})

	for _, name := range []agent.ToolName{editName, writeName, moveName} {
		var tc *agent.ToolCall
		if name == moveName {
			tc = moveCall(t, p, filepath.Join("shared", "dst.md"))
		} else {
			tc = pathCall(t, name, p)
		}

		_, err := h.Apply(context.Background(), tc)

		assertDenied(t, err)
	}
}

func TestApply_MoveVerifiesBothPaths(t *testing.T) {
	src := filepath.Join("shared", "file.md")
	dst := filepath.Join("forbidden", "file.md")
	h := newHook(t, hooks.Rule{Pattern: filepath.Join("shared", "**"), Access: hooks.Write})

	_, err := h.Apply(context.Background(), moveCall(t, src, dst))

	assertDenied(t, err)
}

func TestApply_FirstMatchWins(t *testing.T) {
	p := filepath.Join("shared", "sub", "file.md")

	denied := newHook(t,
		hooks.Rule{Pattern: filepath.Join("shared", "**"), Access: hooks.No},
		hooks.Rule{Pattern: filepath.Join("shared", "sub", "**"), Access: hooks.Write},
	)
	_, err := denied.Apply(context.Background(), pathCall(t, writeName, p))
	assertDenied(t, err)

	allowed := newHook(t,
		hooks.Rule{Pattern: filepath.Join("shared", "sub", "**"), Access: hooks.Write},
		hooks.Rule{Pattern: filepath.Join("shared", "**"), Access: hooks.No},
	)
	got, err := allowed.Apply(context.Background(), pathCall(t, writeName, p))
	assertAllowed(t, got, err)
}

func TestApply_RejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("data"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks are not supported: %v", err)
	}

	h := newHook(t, hooks.Rule{Pattern: filepath.Join(dir, "**"), Access: hooks.Read})

	_, err := h.Apply(context.Background(), pathCall(t, readName, link))

	assertDenied(t, err)
}

func TestApply_BadPath(t *testing.T) {
	h := newHook(t, hooks.Rule{Pattern: "**", Access: hooks.Write})

	_, err := h.Apply(context.Background(), pathCall(t, readName, "a\x00b"))

	assertDenied(t, err)
}
