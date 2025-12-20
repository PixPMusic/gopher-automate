package actions

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// AppleScriptHandler handles AppleScript execution logic
type AppleScriptHandler struct{}

func (h *AppleScriptHandler) IsSupported() bool {
	return runtime.GOOS == "darwin"
}

func (h *AppleScriptHandler) Execute(code string) (string, error) {
	if !h.IsSupported() {
		return "", fmt.Errorf("AppleScript is only supported on macOS")
	}

	cmd := exec.Command("osascript", "-e", code)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return "", fmt.Errorf("AppleScript error: %s", strings.TrimSpace(errMsg))
		}
		return "", fmt.Errorf("AppleScript execution failed: %v", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (h *AppleScriptHandler) Validate(code string) error {
	if !h.IsSupported() {
		return fmt.Errorf("AppleScript validation only available on macOS")
	}

	// Use osacompile to check syntax
	cmd := exec.Command("osacompile", "-o", "/dev/null", "-e", code)
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
