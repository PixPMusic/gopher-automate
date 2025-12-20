package actions

import (
	"fmt"

	"github.com/PixPMusic/gopher-automate/internal/midi"
)

// Executor handles action execution with platform-specific logic
type Executor struct {
	handlers map[ActionType]ActionHandler
}

// NewExecutor creates a new action executor
func NewExecutor(midiManager *midi.Manager) *Executor {
	return &Executor{
		handlers: map[ActionType]ActionHandler{
			ActionTypeAppleScript:  &AppleScriptHandler{},
			ActionTypeShellCommand: &ShellHandler{},
			ActionTypeSleep:        &SleepHandler{},
			ActionTypeMidi:         NewMidiHandler(midiManager),
		},
	}
}

// Execute runs an action based on its type
// Returns output and error (error if type not supported on current platform)
func (e *Executor) Execute(action *Action) (string, error) {
	if action == nil {
		return "", fmt.Errorf("action is nil")
	}

	handler, ok := e.handlers[action.Type]
	if !ok {
		return "", fmt.Errorf("unknown action type: %s", action.Type)
	}

	return handler.Execute(action.Code)
}

// ValidateAppleScript checks AppleScript syntax without executing (darwin only)
func (e *Executor) ValidateAppleScript(code string) error {
	handler, ok := e.handlers[ActionTypeAppleScript]
	if !ok {
		return fmt.Errorf("AppleScript handler not found")
	}
	return handler.Validate(code)
}

// ValidateShellCommand performs basic shell syntax validation
func (e *Executor) ValidateShellCommand(code string) error {
	handler, ok := e.handlers[ActionTypeShellCommand]
	if !ok {
		return fmt.Errorf("Shell handler not found")
	}
	return handler.Validate(code)
}

// CanExecuteAppleScript returns true if AppleScript is supported on this platform
func (e *Executor) CanExecuteAppleScript() bool {
	handler, ok := e.handlers[ActionTypeAppleScript]
	if !ok {
		return false
	}
	return handler.IsSupported()
}

// GetShellName returns the name of the shell used on this platform
func (e *Executor) GetShellName() string {
	handler, ok := e.handlers[ActionTypeShellCommand]
	if !ok {
		return "shell"
	}
	if shellHandler, ok := handler.(*ShellHandler); ok {
		return shellHandler.GetShellName()
	}
	return "shell"
}
