// Copyright (c) PlaneOpsCc
// SPDX-License-Identifier: MPL-2.0

package passcli_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/planeopscc/terraform-provider-protonpass/internal/passcli"
)

func TestCLIError_Error(t *testing.T) {
	err := &passcli.CLIError{Args: []string{"vault", "list"}, ExitCode: 1, Stderr: "failed"}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("expected 'failed' in error, got %q", err.Error())
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		stderr string
		want   bool
	}{
		{"not found", true},
		{"does not exist", true},
		{"no such", true},
		{"unrelated error", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.stderr, func(t *testing.T) {
			err := &passcli.CLIError{Stderr: tt.stderr}
			if got := passcli.IsNotFound(err); got != tt.want {
				t.Errorf("IsNotFound(%q) = %v, want %v", tt.stderr, got, tt.want)
			}
		})
	}
}

func TestIsUnsupportedCommand(t *testing.T) {
	tests := []struct {
		stderr string
		want   bool
	}{
		{"error: unrecognized subcommand 'test'", true},
		{"error: unrecognised subcommand 'test'", true},
		{"error: invalid subcommand", true},
		{"error: unexpected argument '--show-secrets' found", true},
		{"session expired", false},
		{"item not found", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.stderr, func(t *testing.T) {
			err := &passcli.CLIError{ExitCode: 2, Stderr: tt.stderr}
			if got := passcli.IsUnsupportedCommand(err); got != tt.want {
				t.Errorf("IsUnsupportedCommand(%q) = %v, want %v", tt.stderr, got, tt.want)
			}
		})
	}
}

// Only a CLIError carries stderr; a bare error whose text happens to match
// must not be mistaken for one.
func TestIsUnsupportedCommand_NonCLIError(t *testing.T) {
	if passcli.IsUnsupportedCommand(errors.New("unrecognized subcommand 'test'")) {
		t.Error("expected false for a non-CLIError")
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		stderr string
		want   bool
	}{
		{"not logged in", true},
		{"session expired", true},
		{"authentication", true},
		{"unauthorized", true},
		{"unrelated", false},
	}
	for _, tt := range tests {
		t.Run(tt.stderr, func(t *testing.T) {
			err := &passcli.CLIError{Stderr: tt.stderr}
			if got := passcli.IsAuthError(err); got != tt.want {
				t.Errorf("IsAuthError(%q) = %v, want %v", tt.stderr, got, tt.want)
			}
		})
	}
}
