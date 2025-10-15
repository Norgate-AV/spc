package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Norgate-AV/spc/internal/config"
)

// mockCommander implements Commander interface for testing
type mockCommander struct {
	runFunc func() error
}

func (m *mockCommander) Run() error {
	return m.runFunc()
}

func TestCommandBuilder_BuildCommandArgs(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		files       []string
		wantArgs    []string
		wantErr     bool
		errContains string
	}{
		{
			name: "single file, single target series",
			config: &config.Config{
				Target:       "3",
				CompilerPath: "C:/SPlusCC.exe",
			},
			files: []string{"test.usp"},
			wantArgs: func() []string {
				absPath, _ := filepath.Abs("test.usp")
				return []string{"/target", "series3", "/rebuild", absPath}
			}(),
			wantErr: false,
		},
		{
			name: "multiple files, multiple target series",
			config: &config.Config{
				Target:       "234",
				CompilerPath: "C:/SPlusCC.exe",
			},
			files: []string{"file1.usp", "file2.usl"},
			wantArgs: func() []string {
				abs1, _ := filepath.Abs("file1.usp")
				abs2, _ := filepath.Abs("file2.usl")
				return []string{"/target", "series2", "series3", "series4", "/rebuild", abs1, abs2}
			}(),
			wantErr: false,
		},
		{
			name: "with output file",
			config: &config.Config{
				Target:       "4",
				CompilerPath: "C:/SPlusCC.exe",
				OutputFile:   "output.log",
			},
			files: []string{"test.usp"},
			wantArgs: func() []string {
				absPath, _ := filepath.Abs("test.usp")
				return []string{"/target", "series4", "/rebuild", absPath, "/out", "output.log"}
			}(),
			wantErr: false,
		},
		{
			name: "with silent flag",
			config: &config.Config{
				Target:       "3",
				CompilerPath: "C:/SPlusCC.exe",
				Silent:       true,
			},
			files: []string{"test.usp"},
			wantArgs: func() []string {
				absPath, _ := filepath.Abs("test.usp")
				return []string{"/target", "series3", "/rebuild", absPath, "/silent"}
			}(),
			wantErr: false,
		},
		{
			name: "with user folders",
			config: &config.Config{
				Target:       "3",
				CompilerPath: "C:/SPlusCC.exe",
				UserFolders:  []string{"C:/MyIncludes", "C:/MoreIncludes"},
			},
			files: []string{"test.usp"},
			wantArgs: func() []string {
				absPath, _ := filepath.Abs("test.usp")
				return []string{
					"/target", "series3",
					"/usersplusfolder", "C:/MyIncludes",
					"/usersplusfolder", "C:/MoreIncludes",
					"/rebuild", absPath,
				}
			}(),
			wantErr: false,
		},
		{
			name: "with empty user folder (should skip)",
			config: &config.Config{
				Target:       "3",
				CompilerPath: "C:/SPlusCC.exe",
				UserFolders:  []string{"", "C:/MyIncludes"},
			},
			files: []string{"test.usp"},
			wantArgs: func() []string {
				absPath, _ := filepath.Abs("test.usp")
				return []string{
					"/target", "series3",
					"/usersplusfolder", "C:/MyIncludes",
					"/rebuild", absPath,
				}
			}(),
			wantErr: false,
		},
		{
			name: "all options combined",
			config: &config.Config{
				Target:       "34",
				CompilerPath: "C:/SPlusCC.exe",
				UserFolders:  []string{"C:/Include1"},
				OutputFile:   "build.log",
				Silent:       true,
			},
			files: []string{"main.usp", "helper.usl"},
			wantArgs: func() []string {
				abs1, _ := filepath.Abs("main.usp")
				abs2, _ := filepath.Abs("helper.usl")
				return []string{
					"/target", "series3", "series4",
					"/usersplusfolder", "C:/Include1",
					"/rebuild", abs1, abs2,
					"/out", "build.log",
					"/silent",
				}
			}(),
			wantErr: false,
		},
		{
			name: "invalid target series",
			config: &config.Config{
				Target:       "invalid",
				CompilerPath: "C:/SPlusCC.exe",
			},
			files:       []string{"test.usp"},
			wantErr:     true,
			errContains: "invalid target series",
		},
		{
			name: "empty target",
			config: &config.Config{
				Target:       "",
				CompilerPath: "C:/SPlusCC.exe",
			},
			files:       []string{"test.usp"},
			wantErr:     true,
			errContains: "invalid target series",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewCommandBuilder()
			args, err := cb.BuildCommandArgs(tt.config, tt.files)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}

func TestCommandBuilder_ExecuteCommand_Success(t *testing.T) {
	cb := NewCommandBuilder()

	// Mock exec.Command to return success
	cb.execCommand = func(name string, args ...string) Commander {
		return &mockCommander{
			runFunc: func() error {
				return nil
			},
		}
	}

	err := cb.ExecuteCommand("C:/SPlusCC.exe", []string{"/target", "series3"})
	assert.NoError(t, err)
}

func TestCommandBuilder_ExecuteCommand_CompilerSuccess_ExitCode116(t *testing.T) {
	cb := NewCommandBuilder()

	// Mock exec.Command to return exit code 116 (success with warnings)
	cb.execCommand = func(name string, args ...string) Commander {
		return &mockCommander{
			runFunc: func() error {
				return &exec.ExitError{ProcessState: &os.ProcessState{}}
			},
		}
	}

	// We need to mock the exit code check
	// For now, this test documents the expected behavior
	// In real execution, exit code 116 should be treated as success
}

func TestCommandBuilder_ExecuteCommand_CompilerError(t *testing.T) {
	cb := NewCommandBuilder()
	capturedStderr := ""

	// Mock exec.Command to return exit code 106 (compile errors)
	cb.execCommand = func(name string, args ...string) Commander {
		cmd := exec.Command("cmd", "/c", "exit", "106")
		return cmd
	}

	err := cb.ExecuteCommand("C:/SPlusCC.exe", []string{"/target", "series3"})

	// Should return error
	assert.Error(t, err)

	// Error should be an ExitError
	var exitErr *exec.ExitError
	if assert.ErrorAs(t, err, &exitErr) {
		assert.Equal(t, 106, exitErr.ExitCode())
	}

	// Note: In real execution, descriptive message is printed to stderr
	_ = capturedStderr
}

func TestCommandBuilder_ExecuteCommand_NonExitError(t *testing.T) {
	cb := NewCommandBuilder()

	// Mock exec.Command to return a non-exit error
	cb.execCommand = func(name string, args ...string) Commander {
		return &mockCommander{
			runFunc: func() error {
				return fmt.Errorf("command not found")
			},
		}
	}

	err := cb.ExecuteCommand("nonexistent.exe", []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command not found")
}

func TestCommandBuilder_PrintBuildInfo(t *testing.T) {
	cb := NewCommandBuilder()
	cfg := &config.Config{
		CompilerPath: "C:/SPlusCC.exe",
		Target:       "34",
		OutputFile:   "build.log",
		UserFolders:  []string{"C:/Include"},
	}

	series := []string{"series3", "series4"}
	args := []string{"test.usp"}
	cmdArgs := []string{"/target", "series3", "series4", "/rebuild", "test.usp"}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cb.PrintBuildInfo(cfg, series, args, cmdArgs)

	w.Close()
	os.Stdout = oldStdout

	// Read from pipe
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify output contains key information
	assert.Contains(t, output, "C:/SPlusCC.exe")
	assert.Contains(t, output, "34")
	assert.Contains(t, output, "series3")
	assert.Contains(t, output, "series4")
	assert.Contains(t, output, "test.usp")
	assert.Contains(t, output, "build.log")
	assert.Contains(t, output, "C:/Include")
}

func TestNewCommandBuilder(t *testing.T) {
	cb := NewCommandBuilder()
	assert.NotNil(t, cb)
	assert.NotNil(t, cb.execCommand)
}
