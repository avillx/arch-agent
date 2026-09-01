package shell

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	rt "runtime"
	"strings"
	"time"
)

var defaultTimeout = 30 * time.Second

type ShellToolServer struct {
	*tools.BuildInToolServer
}

func NewShellToolServer(defaultCWD string, env ...string) *ShellToolServer {
	return &ShellToolServer{
		BuildInToolServer: tools.NewBuildInToolServer(
			&ShellTool{
				env:        env,
				defaultCWD: defaultCWD,
			},
		),
	}
}

func (t *ShellToolServer) Instruction() string {
	return `## Shell Tool

- Current OS is "` + rt.GOOS + `." 
- Your shell is "` + shell + `"
- Each call runs in a fresh shell session — no state persists between calls
- Default timeout: 30s. Set "timeout" for longer operations (builds, tests)
- Use "cwd" parameter instead of cd within commands
- Combine operations with pipes, redirections, and heredocs
- Non-zero exit codes return error info with output; timed-out commands are terminated
- NEVER try to read env directly, just ask user env name if you need one.`
}

var _ agent.Tool = (*ShellTool)(nil)

type ShellTool struct {
	env        []string
	defaultCWD string
}

func (t *ShellTool) Name() agent.ToolName {
	return "shell"
}

// this is edge timeout, sets in runtime
func (t *ShellTool) TimeOut() time.Duration {
	return 15 * time.Minute
}

func (t *ShellTool) Description() string {
	return "access to shell"
}

func (t *ShellTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:        "command",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Full shell command for execution",
		},
		{
			Name:        "cwd",
			Required:    false,
			Type:        agent.TypeString,
			Description: "Working directory (default \".\")",
		},
		{
			Name:        "timeout",
			Required:    false,
			Type:        agent.TypeNumber,
			Description: "Timeout in seconds",
		},
	}
}

const waitDelayAfterShellExit = 5 * time.Second

func (t *ShellTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Command string        `json:"command"`
		Timeout time.Duration `json:"timeout,omitempty"`
		Cwd     string        `json:"cwd,omitempty"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	// envirement variables for agent shell calls
	envs := os.Environ()
	envs = append(envs, t.env...)
	envs = append(envs, metadataToEnv(ctx)...)

	cmd := exec.Command(shell, append(shellAttributes, args.Command)...)
	cmd.Env = envs
	cmd.Dir = t.defaultCWD
	cmd.SysProcAttr = platformSpecificSysProcAttr()
	cmd.WaitDelay = waitDelayAfterShellExit

	if args.Cwd != "" {
		cmd.Dir = args.Cwd
	}

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Start(); err != nil {
		return tools.Result(fmt.Sprintf("Error starting command: %s", err)), err
	}

	pg, err := createProcessGroup(cmd.Process)
	if err != nil {
		// Successfully started the child but couldn't install it in its own
		// process group: clean it up before bailing out.
		reapSpawnedChild(cmd, pg)
		return tools.Result(fmt.Sprintf("Error creating process group: %s", err)), err
	}

	//
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// timeout
	to := defaultTimeout
	if args.Timeout > 0 {
		to = args.Timeout * time.Second
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(to))
	defer cancel()

	// wait for
	var cmdErr error
	select {
	case <-ctx.Done():
		_ = kill(cmd.Process, pg)

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	case cmdErr = <-done:
	}

	formattedOutput := formatCommandOutput(ctx, cmdErr, output.String(), to)
	return tools.Result(formattedOutput), nil
}

// helpers
func formatCommandOutput(ctx context.Context, err error, output string, timeout time.Duration) string {
	var sb strings.Builder

	sb.WriteString(output)

	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.Canceled) {
			sb.WriteString("Command cancelled\n")
		}
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			fmt.Fprintf(&sb, "Command timed out after %v", timeout)
		}
	}

	if err != nil {
		fmt.Fprintf(&sb, "Error executing command: %s", err)
	}

	return cmp.Or(strings.TrimSpace(sb.String()), "<no output>")
}

func reapSpawnedChild(cmd *exec.Cmd, pg *processGroup) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = kill(cmd.Process, pg)

	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}
}

func metadataToEnv(ctx context.Context) []string {
	agentID := tools.MustAgentID(ctx)
	sessionID := tools.MustSessionID(ctx)

	return []string{
		fmt.Sprintf("AGENT_ID=%s", agentID),
		fmt.Sprintf("AGENT_SESSION_ID=%s", sessionID),
	}
}
