package actions

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ShellHandler handles Shell Command execution logic
type ShellHandler struct{}

func (h *ShellHandler) IsSupported() bool {
	// Shell is supported on all major platforms (PowerShell on Windows, Bash/Zsh on Unix)
	return true
}

func (h *ShellHandler) Execute(code string) (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use PowerShell on Windows
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", code)
	case "darwin", "linux":
		// Use default shell (typically bash or zsh) on Unix-like systems
		shell := "/bin/bash"
		// Try to use zsh if available on macOS
		if runtime.GOOS == "darwin" {
			if _, err := exec.LookPath("zsh"); err == nil {
				shell = "/bin/zsh"
			}
		}
		cmd = exec.Command(shell, "-c", code)
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return stdout.String(), fmt.Errorf("shell error: %s", strings.TrimSpace(errMsg))
		}
		return stdout.String(), fmt.Errorf("shell execution failed: %v", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (h *ShellHandler) Validate(code string) error {
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("empty command")
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// PowerShell syntax check using -NoExecute (parse only)
		// This doesn't actually exist in PowerShell, so we'll just do basic checks
		if strings.Contains(code, "\x00") {
			return fmt.Errorf("command contains null bytes")
		}
		return nil
	case "darwin", "linux":
		// Use bash -n for syntax checking (parses but doesn't execute)
		cmd = exec.Command("/bin/bash", "-n", "-c", code)
	default:
		return nil // Skip validation on unknown platforms
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return fmt.Errorf("syntax error: %s", strings.TrimSpace(errMsg))
		}
		return fmt.Errorf("validation failed: %v", err)
	}

	return nil
}

// GetShellName returns the name of the shell used on this platform
func (h *ShellHandler) GetShellName() string {
	switch runtime.GOOS {
	case "windows":
		return "PowerShell"
	case "darwin":
		return "zsh"
	case "linux":
		return "bash"
	default:
		return "shell"
	}
}
