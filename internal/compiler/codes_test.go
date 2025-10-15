package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		want     bool
	}{
		{
			name:     "exit code 0 is success",
			exitCode: 0,
			want:     true,
		},
		{
			name:     "exit code 116 is success (with warnings)",
			exitCode: 116,
			want:     true,
		},
		{
			name:     "exit code 100 is failure",
			exitCode: 100,
			want:     false,
		},
		{
			name:     "exit code 106 is failure (compile errors)",
			exitCode: 106,
			want:     false,
		},
		{
			name:     "exit code 1 is failure",
			exitCode: 1,
			want:     false,
		},
		{
			name:     "exit code 999 is failure",
			exitCode: 999,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSuccess(tt.exitCode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		want     string
	}{
		{
			name:     "exit code 0",
			exitCode: 0,
			want:     "Success",
		},
		{
			name:     "exit code 100",
			exitCode: 100,
			want:     "General failure",
		},
		{
			name:     "exit code 106 - compile errors",
			exitCode: 106,
			want:     "Compile errors",
		},
		{
			name:     "exit code 107 - link errors",
			exitCode: 107,
			want:     "Link errors",
		},
		{
			name:     "exit code 116 - success with errors",
			exitCode: 116,
			want:     "The system.CodeDom.Compiler finished successfully, but with errors",
		},
		{
			name:     "exit code 112 - GNU not installed",
			exitCode: 112,
			want:     "GNU not installed",
		},
		{
			name:     "exit code 130 - signing error",
			exitCode: 130,
			want:     "Error found while signing. Unable to cleanup unsigned assembly.",
		},
		{
			name:     "unknown exit code",
			exitCode: 999,
			want:     "Unknown error",
		},
		{
			name:     "negative exit code",
			exitCode: -1,
			want:     "Unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetErrorMessage(tt.exitCode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestErrorCodes_Coverage(t *testing.T) {
	// Verify all error codes in the map are accessible
	knownCodes := []int{
		0, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110,
		111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122,
		123, 124, 125, 126, 127, 128, 129, 130,
	}

	for _, code := range knownCodes {
		msg := GetErrorMessage(code)
		assert.NotEqual(t, "Unknown error", msg, "Code %d should have a message", code)
		assert.NotEmpty(t, msg, "Code %d should have a non-empty message", code)
	}
}
