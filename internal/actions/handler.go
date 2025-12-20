package actions

// ActionHandler defines the interface for executing and validating actions
type ActionHandler interface {
	// Execute runs the code and returns output or error
	Execute(code string) (string, error)

	// Validate checks the syntax of the code
	Validate(code string) error

	// IsSupported returns true if the handler can run on the current platform
	IsSupported() bool
}
