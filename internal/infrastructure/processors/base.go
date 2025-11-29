package processors

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/histopathai/image-processing-service/pkg/errors"
)

// BaseProcessor provides common functionality for CLI-based processors
type BaseProcessor struct {
	logger     *slog.Logger
	binaryName string
}

// NewBaseProcessor creates a new base processor instance
func NewBaseProcessor(logger *slog.Logger, binaryName string) *BaseProcessor {
	return &BaseProcessor{
		logger:     logger,
		binaryName: binaryName,
	}
}

// VerifyBinary checks if the binary exists in system PATH
func (p *BaseProcessor) VerifyBinary() error {
	_, err := exec.LookPath(p.binaryName)
	if err != nil {
		return errors.NewConfigurationError("executable not found in PATH").
			WithContext("binary", p.binaryName)
	}
	return nil
}

func (p *BaseProcessor) Execute(ctx context.Context, args []string, timeoutMinutes int) (*CommandResult, error) {
	if timeoutMinutes <= 0 {
		return nil, errors.NewValidationError("timeout must be positive").
			WithContext("timeout_minutes", timeoutMinutes)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.binaryName, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	p.logCommandStart(args, timeoutMinutes)

	err := cmd.Run()

	return p.handleCommandResult(ctx, cmd, stdout, stderr, err, timeoutMinutes)
}

func (p *BaseProcessor) ExecuteWithInput(ctx context.Context, args []string, input io.Reader, timeoutMinutes int) (*CommandResult, error) {
	if timeoutMinutes <= 0 {
		return nil, errors.NewValidationError("timeout must be positive").
			WithContext("timeout_minutes", timeoutMinutes)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.binaryName, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = input
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	p.logCommandStart(args, timeoutMinutes)

	err := cmd.Run()

	return p.handleCommandResult(ctx, cmd, stdout, stderr, err, timeoutMinutes)
}

func (p *BaseProcessor) ExecuteToFile(ctx context.Context, args []string, outputFilePath string, timeoutMinutes int) (*CommandResult, error) {
	if timeoutMinutes <= 0 {
		return nil, errors.NewValidationError("timeout must be positive").
			WithContext("timeout_minutes", timeoutMinutes)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)
	defer cancel()

	file, err := os.Create(outputFilePath)
	if err != nil {
		return nil, errors.WrapStorageError(err, "failed to create output file").
			WithContext("output_file", outputFilePath)
	}
	defer file.Close()

	cmd := exec.CommandContext(ctx, p.binaryName, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(file, &stdout) // Write to both file and buffer
	cmd.Stderr = &stderr

	p.logCommandStart(args, timeoutMinutes)

	err = cmd.Run()

	return p.handleCommandResult(ctx, cmd, stdout, stderr, err, timeoutMinutes)
}

func (p *BaseProcessor) handleCommandResult(ctx context.Context, cmd *exec.Cmd, stdout, stderr bytes.Buffer, err error, timeoutMinutes int) (*CommandResult, error) {
	result := p.createResult(stdout, stderr, err)

	// Check context errors first
	if ctx.Err() == context.DeadlineExceeded {
		p.logger.Error("command timed out",
			"binary", p.binaryName,
			"exit_code", result.ExitCode,
			"timeout_minutes", timeoutMinutes,
			"stderr", stderr.String(),
		)
		return result, errors.WrapTimeoutError(err, fmt.Sprintf("command timed out after %d minutes", timeoutMinutes)).
			WithContext("binary", p.binaryName).
			WithContext("exit_code", result.ExitCode).
			WithContext("stderr", stderr.String())
	}

	if ctx.Err() == context.Canceled {
		p.logger.Warn("command canceled",
			"binary", p.binaryName,
			"exit_code", result.ExitCode,
		)
		return result, errors.New(errors.ErrorTypeCancellation, "command execution canceled").
			WithContext("binary", p.binaryName).
			WithContext("exit_code", result.ExitCode)
	}

	// Handle command execution errors
	if err != nil {
		return result, p.categorizeCommandError(result, err)
	}

	return result, nil
}

func (p *BaseProcessor) categorizeCommandError(result *CommandResult, err error) error {
	exitCode := result.ExitCode
	stderr := result.Stderr

	switch exitCode {
	case 126:
		// Permission or not executable - configuration issue
		p.logger.Error("command permission error",
			"binary", p.binaryName,
			"exit_code", exitCode,
			"stderr", stderr,
		)
		return errors.NewConfigurationError("permission denied or command not executable").
			WithContext("binary", p.binaryName).
			WithContext("exit_code", exitCode).
			WithContext("stderr", stderr)

	case 127:
		// Command not found - configuration issue
		p.logger.Error("command not found",
			"binary", p.binaryName,
			"exit_code", exitCode,
			"stderr", stderr,
		)
		return errors.NewConfigurationError("command not found").
			WithContext("binary", p.binaryName).
			WithContext("exit_code", exitCode).
			WithContext("stderr", stderr)

	case 137:
		// Process killed (SIGKILL) - likely resource issue, retryable
		p.logger.Warn("command killed",
			"binary", p.binaryName,
			"exit_code", exitCode,
			"stderr", stderr,
		)
		return errors.WrapProcessingError(err, "command was killed, possibly due to resource limits").
			WithContext("binary", p.binaryName).
			WithContext("exit_code", exitCode).
			WithContext("stderr", stderr)

	case 143:
		// Terminated (SIGTERM) - retryable
		p.logger.Warn("command terminated",
			"binary", p.binaryName,
			"exit_code", exitCode,
			"stderr", stderr,
		)
		return errors.WrapProcessingError(err, "command was terminated").
			WithContext("binary", p.binaryName).
			WithContext("exit_code", exitCode).
			WithContext("stderr", stderr)

	case 1, 2:
		// General errors - likely bug in command usage
		p.logger.Error("command failed with error",
			"binary", p.binaryName,
			"exit_code", exitCode,
			"stderr", stderr,
		)
		return errors.WrapProcessingError(err, "command execution failed").
			WithContext("binary", p.binaryName).
			WithContext("exit_code", exitCode).
			WithContext("stderr", stderr)

	default:
		// Unknown exit code - treat as processing error
		p.logger.Error("command failed with unknown exit code",
			"binary", p.binaryName,
			"exit_code", exitCode,
			"stderr", stderr,
		)
		return errors.WrapProcessingError(err, fmt.Sprintf("command failed with exit code %d", exitCode)).
			WithContext("binary", p.binaryName).
			WithContext("exit_code", exitCode).
			WithContext("stderr", stderr)
	}
}

func (p *BaseProcessor) createResult(stdout, stderr bytes.Buffer, err error) *CommandResult {
	result := &CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else if err == nil {
		result.ExitCode = 0
	} else {
		result.ExitCode = -1
	}

	return result
}

func (p *BaseProcessor) logCommandStart(args []string, timeoutMinutes int) {
	if p.logger != nil {
		p.logger.Debug("executing command",
			"binary", p.binaryName,
			"args", args,
			"timeout_minutes", timeoutMinutes,
		)
	}
}
