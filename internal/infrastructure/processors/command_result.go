package processors

import (
	"fmt"
	"strings"
)

// CommandResult holds the output of a command execution
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func (r *CommandResult) IsSuccess() bool {
	return r.ExitCode == 0
}

func (r *CommandResult) IsFailure() bool {
	return r.ExitCode != 0
}

func (r *CommandResult) Output() string {
	return strings.TrimSpace(r.Stdout)
}

func (r *CommandResult) ErrorString() string {
	if r.Stderr != "" {
		return strings.TrimSpace(r.Stderr)
	}
	return fmt.Sprintf("command failed with exit code %d", r.ExitCode)
}

// Retryable determines if the command should be retried based on exit code
// This is kept for backward compatibility, but prefer checking error type
func (r *CommandResult) Retryable() bool {
	if r.IsSuccess() {
		return false
	}

	switch r.ExitCode {
	case 126: // Permission problem - configuration issue, not retryable
		return false
	case 127: // Command not found - configuration issue, not retryable
		return false
	case 1, 2: // General/misuse errors - likely bug, not retryable
		return false
	case 137: // SIGKILL - resource issue, retryable
		return true
	case 143: // SIGTERM - terminated, retryable
		return true
	default:
		// Conservative: unknown exit codes are not retryable
		return false
	}
}

// ExitCodeDescription provides human-readable description of exit code
func (r *CommandResult) ExitCodeDescription() string {
	switch r.ExitCode {
	case 0:
		return "success"
	case 1:
		return "general error"
	case 2:
		return "misuse of shell command"
	case 126:
		return "permission denied or not executable"
	case 127:
		return "command not found"
	case 137:
		return "killed (SIGKILL) - likely resource limit"
	case 143:
		return "terminated (SIGTERM)"
	default:
		return fmt.Sprintf("unknown exit code %d", r.ExitCode)
	}
}
