// Copyright (c) PlaneOpsCc
// SPDX-License-Identifier: MPL-2.0

package passcli

import (
	"errors"
	"fmt"
	"strings"
)

// CLIError wraps a pass-cli execution failure.
type CLIError struct {
	Args     []string
	ExitCode int
	Stderr   string
	Err      error
}

func (e *CLIError) Error() string {
	return fmt.Sprintf("pass-cli %v exited with code %d: %s", RedactArgs(e.Args), e.ExitCode, e.Stderr)
}

func (e *CLIError) Unwrap() error {
	return e.Err
}

// IsNotFound returns true if the error indicates a resource was not found.
func IsNotFound(err error) bool {
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		lower := strings.ToLower(cliErr.Stderr)
		return strings.Contains(lower, "not found") ||
			strings.Contains(lower, "does not exist") ||
			strings.Contains(lower, "no such")
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

// IsUnsupportedCommand returns true if the error indicates that the installed
// pass-cli does not recognise the subcommand or flag it was invoked with.
// clap (pass-cli's argument parser) reports this on stderr and exits non-zero,
// which is otherwise indistinguishable from a real failure. Detecting it lets
// the client fall back to an older command spelling on older CLIs.
func IsUnsupportedCommand(err error) bool {
	var cliErr *CLIError
	if !errors.As(err, &cliErr) {
		return false
	}
	lower := strings.ToLower(cliErr.Stderr)
	return strings.Contains(lower, "unrecognized subcommand") ||
		strings.Contains(lower, "unrecognised subcommand") ||
		strings.Contains(lower, "invalid subcommand") ||
		strings.Contains(lower, "unexpected argument")
}

// IsAuthError returns true if the error indicates an authentication problem.
func IsAuthError(err error) bool {
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		lower := strings.ToLower(cliErr.Stderr)
		return strings.Contains(lower, "not logged in") ||
			strings.Contains(lower, "session expired") ||
			strings.Contains(lower, "authentication") ||
			strings.Contains(lower, "unauthorized")
	}
	return false
}
