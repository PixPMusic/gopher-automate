package startup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Enable registers the application to launch at system startup
func Enable() error {
	switch runtime.GOOS {
	case "darwin":
		return enableMacOS()
	case "linux":
		return enableLinux()
	case "windows":
		return enableWindows()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Disable removes the application from system startup
func Disable() error {
	switch runtime.GOOS {
	case "darwin":
		return disableMacOS()
	case "linux":
		return disableLinux()
	case "windows":
		return disableWindows()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// IsEnabled checks if the application is registered for startup
func IsEnabled() bool {
	switch runtime.GOOS {
	case "darwin":
		return isEnabledMacOS()
	case "linux":
		return isEnabledLinux()
	case "windows":
		return isEnabledWindows()
	default:
		return false
	}
}

// --- macOS Implementation ---

const macOSPlistName = "com.gopher-automate.plist"

func macOSPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", macOSPlistName)
}

func enableMacOS() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.gopher-automate</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`, execPath)

	// Ensure LaunchAgents directory exists
	dir := filepath.Dir(macOSPlistPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(macOSPlistPath(), []byte(plistContent), 0644)
}

func disableMacOS() error {
	path := macOSPlistPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Already disabled
	}
	return os.Remove(path)
}

func isEnabledMacOS() bool {
	_, err := os.Stat(macOSPlistPath())
	return err == nil
}

// --- Linux Implementation ---

const linuxDesktopName = "gopher-automate.desktop"

func linuxDesktopPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "autostart", linuxDesktopName)
}

func enableLinux() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=GopherAutomate
Exec=%s
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
`, execPath)

	// Ensure autostart directory exists
	dir := filepath.Dir(linuxDesktopPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(linuxDesktopPath(), []byte(desktopContent), 0644)
}

func disableLinux() error {
	path := linuxDesktopPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Already disabled
	}
	return os.Remove(path)
}

func isEnabledLinux() bool {
	_, err := os.Stat(linuxDesktopPath())
	return err == nil
}

// --- Windows Implementation ---

const windowsRegistryKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
const windowsAppName = "GopherAutomate"

func enableWindows() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	// Use reg.exe to add the registry key
	cmd := exec.Command("reg", "add", windowsRegistryKey,
		"/v", windowsAppName,
		"/t", "REG_SZ",
		"/d", execPath,
		"/f")
	return cmd.Run()
}

func disableWindows() error {
	cmd := exec.Command("reg", "delete", windowsRegistryKey,
		"/v", windowsAppName,
		"/f")
	output, err := cmd.CombinedOutput()
	// Ignore error if the key doesn't exist
	if err != nil && !strings.Contains(string(output), "The system was unable to find the specified registry key or value") {
		return err
	}
	return nil
}

func isEnabledWindows() bool {
	cmd := exec.Command("reg", "query", windowsRegistryKey,
		"/v", windowsAppName)
	err := cmd.Run()
	return err == nil
}
