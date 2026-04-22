package internal

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Agent represents an AI assistant accessible via command line.
// Each agent has its own binary and configurable arguments.
type Agent struct {
	// Friendly name, e.g.: "Gemini", "Kiro", "Copilot"
	Name string `json:"name"`

	// Binary to execute, e.g.: "gemini", "kiro", "gh"
	Command string `json:"command"`

	// Arguments passed to the binary.
	// Use {prompt} as a placeholder — it will be replaced with the actual text.
	// Example: ["copilot", "suggest", "-t", "general", "{prompt}"]
	Args []string `json:"args"`

	// Timeout in seconds. Default: 90s if left as 0.
	TimeoutSeconds int `json:"timeout_seconds"`

	// If true, sends the prompt via stdin instead of substituting in Args.
	// Useful for CLIs that read from standard input.
	UseStdin bool `json:"use_stdin"`
}

// Response is an agent's response to a prompt.
type Response struct {
	Agent   *Agent // which agent generated this response
	Content string // response text (empty if there was an error)
	Err     error  // error, if any (nil = success)
	Letter  string // "A", "B", or "C" — assigned by the council
}

// Ask sends a prompt to the agent and returns the response.
// Runs the binary as a subprocess and captures stdout.
func (a *Agent) Ask(prompt string) Response {
	// Set timeout (minimum 10s, default 90s)
	timeoutSec := a.TimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = 90
	}
	timeout := time.Duration(timeoutSec) * time.Second

	// Create a context with timeout — ensures the process is killed if it takes too long
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Replace the {prompt} placeholder in each argument
	args := make([]string, len(a.Args))
	for i, arg := range a.Args {
		args[i] = strings.ReplaceAll(arg, "{prompt}", prompt)
	}

	// Create the command
	cmd := exec.CommandContext(ctx, a.Command, args...)

	// Capture stdout and stderr separately
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// If UseStdin=true, send the prompt via standard input
	// Useful when {prompt} is not in the args or when the prompt is very long
	if a.UseStdin {
		cmd.Stdin = strings.NewReader(prompt)
	}

	// Run the process and wait for it to finish
	if err := cmd.Run(); err != nil {
		// Timeout is the most common error — give a specific message
		if ctx.Err() == context.DeadlineExceeded {
			return Response{
				Agent: a,
				Err:   fmt.Errorf("timeout after %v — try increasing timeout_seconds in config.json", timeout),
			}
		}
		// Other errors: include stderr to help with debugging
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return Response{
			Agent: a,
			Err:   fmt.Errorf("failed to run %q: %s", a.Command, errMsg),
		}
	}

	content := strings.TrimSpace(stdout.String())
	if content == "" {
		return Response{
			Agent: a,
			Err:   fmt.Errorf("%s returned an empty response — check if the command is correct", a.Name),
		}
	}

	return Response{
		Agent:   a,
		Content: content,
	}
}
