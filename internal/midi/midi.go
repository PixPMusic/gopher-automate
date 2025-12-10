package midi

import (
	"fmt"
	"sync"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // Register rtmidi driver
)

// DeviceType represents the type of device
type DeviceType string

const (
	DeviceTypeClassic  DeviceType = "classic"  // Launchpad S - no special programmer mode
	DeviceTypeColorful DeviceType = "colorful" // Launchpad Mini Mk3 - requires SysEx
)

// PadColor represents an RGB color for a pad
type PadColor struct {
	R, G, B uint8 // 0-127 for each channel
}

// PadMapping describes how to address a pad on a specific device
type PadMapping struct {
	IsCC     bool  // true = Control Change, false = Note
	Number   uint8 // CC number or Note number
	Exists   bool  // false if this pad doesn't exist on the device
	LEDIndex uint8 // For Mini Mk3 SysEx, the LED index
}

// DeviceLayout maps grid positions (row, col) to device-specific pad addresses
type DeviceLayout interface {
	GetPadMapping(row, col int) PadMapping
	GetGridSize() (rows, cols int)
}

// LaunchpadSLayout implements DeviceLayout for Launchpad S
type LaunchpadSLayout struct{}

func (l *LaunchpadSLayout) GetGridSize() (int, int) {
	return 9, 9 // 8x8 grid + 1 row of scene launch + 1 col doesn't exist at (8,8)
}

func (l *LaunchpadSLayout) GetPadMapping(row, col int) PadMapping {
	// Launchpad S layout:
	// Top row (row 0): Control Change 104-111 (8 buttons, no 9th)
	// Grid rows 1-8: Notes, each row is offset by 16
	// Row 1: 0-7, Row 2: 16-23, etc.
	// Right column (col 8): Notes 8, 24, 40, 56, 72, 88, 104, 120

	// Position (8, 8) doesn't exist on Launchpad S
	if row == 0 && col == 8 {
		return PadMapping{Exists: false}
	}

	if row == 0 {
		// Top row: Control Change 104 + col
		return PadMapping{
			IsCC:   true,
			Number: uint8(104 + col),
			Exists: true,
		}
	}

	// Grid and right column: Note messages
	// Row 1 = notes 0-8, Row 2 = notes 16-24, etc.
	noteNum := uint8((row-1)*16 + col)
	return PadMapping{
		IsCC:   false,
		Number: noteNum,
		Exists: true,
	}
}

// LaunchpadMiniMk3Layout implements DeviceLayout for Launchpad Mini Mk3
type LaunchpadMiniMk3Layout struct{}

func (l *LaunchpadMiniMk3Layout) GetGridSize() (int, int) {
	return 9, 9 // Full 9x9 grid
}

func (l *LaunchpadMiniMk3Layout) GetPadMapping(row, col int) PadMapping {
	// Launchpad Mini Mk3 programmer mode layout:
	// LED indices: bottom-left is 11, top-right is 99
	// Row formula: LED = (9 - row) * 10 + (col + 1)
	// Top row (row 0): 91-99
	// Bottom row (row 8): 11-19
	ledIndex := uint8((8-row)*10 + col + 11)

	return PadMapping{
		LEDIndex: ledIndex,
		Exists:   true,
	}
}

// GetLayoutForDevice returns the appropriate layout for a device type
func GetLayoutForDevice(deviceType DeviceType) DeviceLayout {
	switch deviceType {
	case DeviceTypeClassic:
		return &LaunchpadSLayout{}
	case DeviceTypeColorful:
		return &LaunchpadMiniMk3Layout{}
	default:
		return &LaunchpadMiniMk3Layout{}
	}
}

// Manager handles MIDI device discovery and management
type Manager struct {
	mu sync.RWMutex
}

// NewManager creates a new MIDI manager
func NewManager() *Manager {
	return &Manager{}
}

// Close cleans up the MIDI driver
func (m *Manager) Close() {
	midi.CloseDriver()
}

// ListInPorts returns the names of available MIDI input ports
func (m *Manager) ListInPorts() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ins := midi.GetInPorts()
	names := make([]string, 0, len(ins))
	for _, in := range ins {
		names = append(names, in.String())
	}
	return names
}

// ListOutPorts returns the names of available MIDI output ports
func (m *Manager) ListOutPorts() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	outs := midi.GetOutPorts()
	names := make([]string, 0, len(outs))
	for _, out := range outs {
		names = append(names, out.String())
	}
	return names
}

// GetInPort returns an input port by name
func (m *Manager) GetInPort(name string) (drivers.In, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ins := midi.GetInPorts()
	for _, in := range ins {
		if in.String() == name {
			return in, nil
		}
	}
	return nil, nil
}

// GetOutPort returns an output port by name
func (m *Manager) GetOutPort(name string) (drivers.Out, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	outs := midi.GetOutPorts()
	for _, out := range outs {
		if out.String() == name {
			return out, nil
		}
	}
	return nil, nil
}

// NoteCallback is called when a Note On/Off event is received
type NoteCallback func(portName string, row, col int, isNoteOn bool)

// StartListening begins listening for MIDI input on the specified port
func (m *Manager) StartListening(inPortName string, deviceType DeviceType, callback NoteCallback) (func(), error) {
	if inPortName == "" {
		return nil, nil
	}

	// Get input port
	inPort, err := m.GetInPort(inPortName)
	if inPort == nil || err != nil {
		return nil, fmt.Errorf("input port not found: %s", inPortName)
	}

	// Create listener
	stop, err := midi.ListenTo(inPort, func(msg midi.Message, timestampms int32) {
		var channel, key, velocity uint8

		switch {
		case msg.GetNoteOn(&channel, &key, &velocity):
			isNoteOn := velocity > 0
			row, col := m.noteToGridPosition(key, deviceType)
			if row >= 0 && col >= 0 {
				callback(inPortName, row, col, isNoteOn)
			}
		case msg.GetNoteOff(&channel, &key, &velocity):
			row, col := m.noteToGridPosition(key, deviceType)
			if row >= 0 && col >= 0 {
				callback(inPortName, row, col, false)
			}
		case msg.GetControlChange(&channel, &key, &velocity):
			// Handle CC for top row buttons on Launchpad S (104-111)
			if deviceType == DeviceTypeClassic && key >= 104 && key <= 111 {
				col := int(key - 104)
				isNoteOn := velocity > 0
				callback(inPortName, 0, col, isNoteOn)
			} else if deviceType == DeviceTypeColorful {
				// Handle CC for top row buttons on Launchpad Mini Mk3 (91-98)
				if key >= 91 && key <= 98 {
					col := int(key - 91)
					isNoteOn := velocity > 0
					callback(inPortName, 0, col, isNoteOn)
				} else if key%10 == 9 && key >= 19 && key <= 89 {
					// Handle CC for right column buttons (19, 29... 89)
					// 19 is bottom right (Row 8), 89 is top right (Row 1)
					row := 8 - int((key-19)/10)
					isNoteOn := velocity > 0
					callback(inPortName, row, 8, isNoteOn)
				}
			}
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to start listening: %w", err)
	}

	return stop, nil
}

// noteToGridPosition converts a MIDI note number to grid row/col
func (m *Manager) noteToGridPosition(note uint8, deviceType DeviceType) (int, int) {
	if deviceType == DeviceTypeClassic {
		// Launchpad S layout: Row 1 = notes 0-8, Row 2 = notes 16-24, etc.
		row := int(note/16) + 1
		col := int(note % 16)
		if row >= 1 && row <= 8 && col >= 0 && col <= 8 {
			return row, col
		}
	} else if deviceType == DeviceTypeColorful {
		// Mini Mk3: LED index = (8-row)*10 + col + 11
		// Invert: row = 8 - (note-11)/10, col = (note-11)%10
		if note >= 11 && note <= 99 {
			row := 8 - int((note-11)/10)
			col := int((note - 11) % 10)
			if row >= 0 && row <= 8 && col >= 0 && col <= 8 {
				return row, col
			}
		}
	}
	return -1, -1
}

// ActivateProgrammerMode sends the appropriate MIDI message to put the device in programmer mode
func (m *Manager) ActivateProgrammerMode(outPortName string, deviceType DeviceType) error {
	if outPortName == "" {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	outPort := m.findOutPort(outPortName)
	if outPort == nil {
		return fmt.Errorf("output port not found: %s", outPortName)
	}

	switch deviceType {
	case DeviceTypeColorful:
		send, err := midi.SendTo(outPort)
		if err != nil {
			return fmt.Errorf("failed to create sender: %w", err)
		}
		// SysEx for programmer mode: 00 20 29 02 0D 0E 01
		sysexContent := []byte{0x00, 0x20, 0x29, 0x02, 0x0D, 0x0E, 0x01}
		if err := send(midi.SysEx(sysexContent)); err != nil {
			return fmt.Errorf("failed to send programmer mode message: %w", err)
		}
	case DeviceTypeClassic:
		// Launchpad S - reset to default state
		send, err := midi.SendTo(outPort)
		if err != nil {
			return fmt.Errorf("failed to create sender: %w", err)
		}
		// Send reset: B0 00 00 (CC 0 value 0)
		if err := send(midi.ControlChange(0, 0, 0)); err != nil {
			return fmt.Errorf("failed to reset Launchpad S: %w", err)
		}
	}

	return nil
}

// SetPadColor sets a pad color using the appropriate method for the device type
func (m *Manager) SetPadColor(outPortName string, deviceType DeviceType, row, col int, color PadColor) error {
	if outPortName == "" {
		return nil
	}

	layout := GetLayoutForDevice(deviceType)
	mapping := layout.GetPadMapping(row, col)

	if !mapping.Exists {
		return nil // Pad doesn't exist on this device
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	outPort := m.findOutPort(outPortName)
	if outPort == nil {
		return fmt.Errorf("output port not found: %s", outPortName)
	}

	send, err := midi.SendTo(outPort)
	if err != nil {
		return fmt.Errorf("failed to create sender: %w", err)
	}

	switch deviceType {
	case DeviceTypeColorful:
		return m.setColorMiniMk3(send, mapping.LEDIndex, color)
	case DeviceTypeClassic:
		return m.setColorLaunchpadS(send, mapping, color)
	}

	return nil
}

// setColorMiniMk3 sets RGB color via SysEx for Mini Mk3
func (m *Manager) setColorMiniMk3(send func(midi.Message) error, ledIndex uint8, color PadColor) error {
	// Apply gamma scaling to make colors more distinct
	// The Mini Mk3 LEDs are already quite linear, but we can enhance saturation
	r := scaleColor(color.R)
	g := scaleColor(color.G)
	b := scaleColor(color.B)

	// SysEx for RGB LED: F0 00 20 29 02 0D 03 03 <led> <r> <g> <b> F7
	sysexContent := []byte{
		0x00, 0x20, 0x29, 0x02, 0x0D, 0x03,
		0x03,     // RGB mode
		ledIndex, // LED index
		r & 0x7F, // R
		g & 0x7F, // G
		b & 0x7F, // B
	}
	return send(midi.SysEx(sysexContent))
}

// scaleColor applies gamma/saturation scaling to make colors more distinct
func scaleColor(value uint8) uint8 {
	if value == 0 {
		return 0
	}
	// Apply a slight curve to enhance color distinction
	// This makes mid-range values more distinct (e.g., orange vs yellow)
	f := float64(value) / 127.0
	// Use a power curve to enhance saturation
	scaled := f * f * 127.0
	if scaled < 1 && value > 0 {
		scaled = 1 // Ensure non-zero input gives non-zero output
	}
	return uint8(scaled)
}

// setColorLaunchpadS sets color via Note/CC velocity for Launchpad S
func (m *Manager) setColorLaunchpadS(send func(midi.Message) error, mapping PadMapping, color PadColor) error {
	// Launchpad S velocity format (from programmer reference):
	// Bit 0-1: Red intensity (0-3)
	// Bit 2: Copy flag (set for immediate update)
	// Bit 3: Clear flag (set for normal operation)
	// Bit 4-5: Green intensity (0-3)
	//
	// Full velocity = 0bGGCCRR where CC = 11 (copy+clear)

	var velocity uint8

	// Check if the color is essentially "off"
	if color.R < 5 && color.G < 5 && color.B < 5 {
		velocity = 0x0C // flags only, no color = off
	} else {
		// Convert RGB to red/green only, mapping blue to maintain brightness
		// Blue leans more toward green since that's closer to blue on the spectrum
		// Blue adds mostly to green, a little to red for brightness
		effectiveR := int(color.R) + int(color.B)/4
		effectiveG := int(color.G) + (int(color.B)*3)/4

		// Clamp to 127
		if effectiveR > 127 {
			effectiveR = 127
		}
		if effectiveG > 127 {
			effectiveG = 127
		}

		// Map 0-127 to 0-3 intensity levels
		redLevel := colorTo4Level(uint8(effectiveR))
		greenLevel := colorTo4Level(uint8(effectiveG))

		// Build velocity: GG CC RR where CC=11 (0x0C shifted)
		// Bits: 5-4=green, 3-2=flags(11), 1-0=red
		velocity = (greenLevel << 4) | 0x0C | redLevel
	}

	if mapping.IsCC {
		return send(midi.ControlChange(0, mapping.Number, velocity))
	}
	return send(midi.NoteOn(0, mapping.Number, velocity))
}

// colorTo4Level converts 0-127 color value to 0-3 intensity for Launchpad S
func colorTo4Level(value uint8) uint8 {
	if value < 32 {
		return 0
	} else if value < 64 {
		return 1
	} else if value < 96 {
		return 2
	}
	return 3
}

// ClearAllPads turns off all LEDs on a device
func (m *Manager) ClearAllPads(outPortName string, deviceType DeviceType) error {
	if outPortName == "" {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	outPort := m.findOutPort(outPortName)
	if outPort == nil {
		return fmt.Errorf("output port not found: %s", outPortName)
	}

	send, err := midi.SendTo(outPort)
	if err != nil {
		return fmt.Errorf("failed to create sender: %w", err)
	}

	switch deviceType {
	case DeviceTypeColorful:
		// Send SysEx to clear all LEDs
		// Using static color 0 for all pads
		sysexContent := []byte{0x00, 0x20, 0x29, 0x02, 0x0D, 0x03}
		for i := 11; i <= 99; i++ {
			if i%10 >= 1 && i%10 <= 9 { // Valid LED indices
				sysexContent = append(sysexContent, 0x00, uint8(i), 0x00) // Static off
			}
		}
		return send(midi.SysEx(sysexContent))

	case DeviceTypeClassic:
		// Reset Launchpad S: B0 00 00
		return send(midi.ControlChange(0, 0, 0))
	}

	return nil
}

func (m *Manager) findOutPort(name string) drivers.Out {
	outs := midi.GetOutPorts()
	for _, out := range outs {
		if out.String() == name {
			return out
		}
	}
	return nil
}
