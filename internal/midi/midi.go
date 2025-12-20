package midi

import (
	"fmt"
	"sync"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // Register rtmidi driver
)

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

// GenericMIDICallback is called for any MIDI message (for inter-app communication)
// msgType: "note", "cc", "program_change"
// channel: 0-15
// number: note/CC/program number (0-127)
// value: velocity/CC value (0-127)
type GenericMIDICallback func(portName string, msgType string, channel, number, value int)

// StartGenericListening listens for all MIDI messages on a port (for inter-app communication)
func (m *Manager) StartGenericListening(inPortName string, callback GenericMIDICallback) (func(), error) {
	if inPortName == "" {
		return nil, nil
	}

	// Get input port
	inPort, err := m.GetInPort(inPortName)
	if inPort == nil || err != nil {
		return nil, fmt.Errorf("input port not found: %s", inPortName)
	}

	// Create listener for all message types
	stop, err := midi.ListenTo(inPort, func(msg midi.Message, timestampms int32) {
		var channel, key, velocity uint8

		switch {
		case msg.GetNoteOn(&channel, &key, &velocity):
			callback(inPortName, "note", int(channel), int(key), int(velocity))
		case msg.GetNoteOff(&channel, &key, &velocity):
			callback(inPortName, "note", int(channel), int(key), 0)
		case msg.GetControlChange(&channel, &key, &velocity):
			callback(inPortName, "cc", int(channel), int(key), int(velocity))
		case msg.GetProgramChange(&channel, &key):
			callback(inPortName, "program_change", int(channel), int(key), 127)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to start generic listening: %w", err)
	}

	return stop, nil
}

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

	device := GetDevice(deviceType)

	// Create listener
	stop, err := midi.ListenTo(inPort, func(msg midi.Message, timestampms int32) {
		row, col, isNoteOn, handled := device.HandleMessage(msg)
		if handled {
			callback(inPortName, row, col, isNoteOn)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to start listening: %w", err)
	}

	return stop, nil
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

	send, err := midi.SendTo(outPort)
	if err != nil {
		return fmt.Errorf("failed to create sender: %w", err)
	}

	device := GetDevice(deviceType)
	return device.ActivateProgrammerMode(send)
}

// SetPadColor sets a pad color using the appropriate method for the device type
func (m *Manager) SetPadColor(outPortName string, deviceType DeviceType, row, col int, color PadColor) error {
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

	device := GetDevice(deviceType)
	return device.SetPadColor(send, row, col, color)
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

	device := GetDevice(deviceType)
	return device.ClearAllPads(send)
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
