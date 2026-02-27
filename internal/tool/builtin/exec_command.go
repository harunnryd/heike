package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	toolcore "github.com/harunnryd/heike/internal/tool"
)

func init() {
	toolcore.RegisterBuiltin("exec_command", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		return &ExecCommandTool{}, nil
	})

	toolcore.RegisterBuiltin("write_stdin", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		return &WriteStdinTool{}, nil
	})
}

// ExecCommandTool executes a shell command.
type ExecCommandTool struct{}

func (t *ExecCommandTool) Name() string {
	return "exec_command"
}

func (t *ExecCommandTool) Description() string {
	return "Execute a shell command. Supports one-shot and interactive session mode."
}

func (t *ExecCommandTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"exec.command",
			"command.run",
			"system.exec",
		},
		Risk: toolcore.RiskHigh,
	}
}

func (t *ExecCommandTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Executable name (legacy structured mode)",
			},
			"args": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Arguments (legacy structured mode)",
			},
			"cmd": map[string]interface{}{
				"type":        "string",
				"description": "Shell command line (Codex-compatible)",
			},
			"tty": map[string]interface{}{
				"type":        "boolean",
				"description": "Start command in interactive session mode",
			},
			"yield_time_ms": map[string]interface{}{
				"type":        "integer",
				"description": "Wait time before returning output (milliseconds)",
			},
			"max_output_tokens": map[string]interface{}{
				"type":        "integer",
				"description": "Approximate maximum output tokens to return",
			},
			"workdir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory for command execution",
			},
			"shell": map[string]interface{}{
				"type":        "string",
				"description": "Shell executable path to use with cmd mode",
			},
			"login": map[string]interface{}{
				"type":        "boolean",
				"description": "When true, use login shell semantics (default true)",
			},
			"justification": map[string]interface{}{
				"type":        "string",
				"description": "Optional human-readable reason for elevated execution",
			},
			"prefix_rule": map[string]interface{}{
				"type":        "array",
				"description": "Optional command prefix recommendation",
				"items":       map[string]interface{}{"type": "string"},
			},
			"sandbox_permissions": map[string]interface{}{
				"type":        "string",
				"description": "Optional execution policy mode hint",
			},
		},
	}
}

func (t *ExecCommandTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var args toolcore.ExecCommandInput
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	cmdText := strings.TrimSpace(args.Cmd)
	command := strings.TrimSpace(args.Command)
	if cmdText == "" && command == "" {
		return nil, fmt.Errorf("cmd or command is required")
	}

	if args.TTY {
		cmd, err := buildExecCommand(nil, args, cmdText, command, true)
		if err != nil {
			return nil, err
		}
		sessionID, err := startExecSession(cmd)
		if err != nil {
			return nil, err
		}

		time.Sleep(normalizeYieldDuration(args.YieldTimeMS))
		session, ok := getExecSession(sessionID)
		if !ok {
			return nil, fmt.Errorf("session not found after start")
		}

		return json.Marshal(map[string]interface{}{
			"session_id": sessionID,
			"output":     session.readNewOutput(args.MaxOutputTokens),
			"running":    session.running(),
			"exit_code":  session.getExitCode(),
		})
	}

	cmd, err := buildExecCommand(ctx, args, cmdText, command, false)
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	exitCode := 0
	result := map[string]interface{}{
		"output":    truncateOutputByTokens(string(output), args.MaxOutputTokens),
		"exit_code": exitCode,
	}
	if err != nil {
		result["error"] = err.Error()
		if cmd.ProcessState != nil {
			result["exit_code"] = cmd.ProcessState.ExitCode()
		} else {
			result["exit_code"] = -1
		}
	}

	return json.Marshal(result)
}

func buildExecCommand(
	ctx context.Context,
	args toolcore.ExecCommandInput,
	cmdText string,
	command string,
	interactive bool,
) (*exec.Cmd, error) {
	workdir := strings.TrimSpace(args.Workdir)

	if cmdText != "" {
		shellPath := resolveExecShell(args.Shell)
		shellArgs := resolveShellArgs(shellPath, execUsesLogin(args.Login))
		shellArgs = append(shellArgs, cmdText)

		var cmd *exec.Cmd
		if interactive {
			cmd = exec.Command(shellPath, shellArgs...)
		} else {
			cmd = exec.CommandContext(ctx, shellPath, shellArgs...)
		}
		if workdir != "" {
			cmd.Dir = workdir
		}
		return cmd, nil
	}

	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	var cmd *exec.Cmd
	if interactive {
		cmd = exec.Command(command, args.Args...)
	} else {
		cmd = exec.CommandContext(ctx, command, args.Args...)
	}
	if workdir != "" {
		cmd.Dir = workdir
	}
	return cmd, nil
}

func execUsesLogin(login *bool) bool {
	if login == nil {
		return true
	}
	return *login
}

func resolveExecShell(shellPath string) string {
	if trimmed := strings.TrimSpace(shellPath); trimmed != "" {
		return trimmed
	}
	if envShell := strings.TrimSpace(os.Getenv("SHELL")); envShell != "" {
		return envShell
	}
	return "sh"
}

func resolveShellArgs(shellPath string, login bool) []string {
	if login && shellSupportsLogin(shellPath) {
		return []string{"-lc"}
	}
	return []string{"-c"}
}

func shellSupportsLogin(shellPath string) bool {
	switch strings.ToLower(filepath.Base(strings.TrimSpace(shellPath))) {
	case "bash", "zsh", "ksh", "mksh":
		return true
	default:
		return false
	}
}

// WriteStdinTool writes to an interactive exec_command session and polls output.
type WriteStdinTool struct{}

func (t *WriteStdinTool) Name() string {
	return "write_stdin"
}

func (t *WriteStdinTool) Description() string {
	return "Write input to an exec_command session and read incremental output."
}

func (t *WriteStdinTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"command.session.write",
			"exec.session.io",
			"system.exec",
		},
		Risk: toolcore.RiskHigh,
	}
}

func (t *WriteStdinTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"session_id": map[string]interface{}{
				"type":        "integer",
				"description": "Session ID returned by exec_command (tty=true)",
			},
			"chars": map[string]interface{}{
				"type":        "string",
				"description": "Characters to write to stdin (optional)",
			},
			"yield_time_ms": map[string]interface{}{
				"type":        "integer",
				"description": "Wait time before polling output (milliseconds)",
			},
			"max_output_tokens": map[string]interface{}{
				"type":        "integer",
				"description": "Approximate maximum output tokens to return",
			},
		},
		"required": []string{"session_id"},
	}
}

func (t *WriteStdinTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	_ = ctx

	var args struct {
		SessionID       int64  `json:"session_id"`
		Chars           string `json:"chars"`
		YieldTimeMS     int    `json:"yield_time_ms"`
		MaxOutputTokens int    `json:"max_output_tokens"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if args.SessionID <= 0 {
		return nil, fmt.Errorf("session_id is required")
	}

	session, ok := getExecSession(args.SessionID)
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	if err := session.write(args.Chars); err != nil {
		return nil, err
	}

	time.Sleep(normalizeYieldDuration(args.YieldTimeMS))
	return json.Marshal(map[string]interface{}{
		"session_id": args.SessionID,
		"output":     session.readNewOutput(args.MaxOutputTokens),
		"running":    session.running(),
		"exit_code":  session.getExitCode(),
	})
}
