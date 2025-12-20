package midi

import "gitlab.com/gomidi/midi/v2"

// Device represents a MIDI device capable of grid-based interaction
type Device interface {
	// ActivateProgrammerMode sends necessary commands to initialize the device
	ActivateProgrammerMode(send func(midi.Message) error) error

	// SetPadColor sets the color of a specific pad
	SetPadColor(send func(midi.Message) error, row, col int, color PadColor) error

	// ClearAllPads clears all pads on the device
	ClearAllPads(send func(midi.Message) error) error

	// HandleMessage parses a MIDI message and returns grid position and state
	// Returns handled=true if the message corresponds to a valid grid event
	HandleMessage(msg midi.Message) (row, col int, isNoteOn bool, handled bool)
}
