package actions

import (
	"encoding/json"
	"fmt"

	internalmidi "github.com/PixPMusic/gopher-automate/internal/midi"
	"gitlab.com/gomidi/midi/v2"
)

// MidiHandler handles MIDI message sending
type MidiHandler struct {
	midiManager *internalmidi.Manager
}

// MidiActionData structure for JSON storage in Code field
type MidiActionData struct {
	DeviceName string `json:"device_name"` // "Any" or specific device name
	MsgType    string `json:"msg_type"`    // "note_on", "note_off", "cc", "pc", "sysex"
	Channel    int    `json:"channel"`     // 1-16
	Note       int    `json:"note"`        // 0-127
	Velocity   int    `json:"velocity"`    // 0-127 (value for CC)
	Program    int    `json:"program"`     // 0-127
	SysEx      string `json:"sysex"`       // Hex string "F0 01 ... F7"
}

func NewMidiHandler(m *internalmidi.Manager) *MidiHandler {
	return &MidiHandler{midiManager: m}
}

func (h *MidiHandler) IsSupported() bool {
	return true
}

func (h *MidiHandler) Execute(code string) (string, error) {
	var data MidiActionData
	if err := json.Unmarshal([]byte(code), &data); err != nil {
		return "", fmt.Errorf("invalid MIDI action data: %v", err)
	}

	// Resolve output port
	// If DeviceName is specific, try to find it. If it's a "Device Config" name, we need to map it to a port.
	// However, the action likely stores the actual Port Name if we populate it from the specific device list,
	// or the Config Name. Let's assume for now the user picks from the Available Output Ports.
	// But `internal/midi/manager.go` has `GetOutPort(name)`.

	// Issue: The device list in the UI might be Config Names (User friendly) or Port Names (System).
	// If we use Port Names, it might break if the device is unplugged/replugged with different ID.
	// But for this MVP, let's assume we use the Port Name string directly as passed.

	if data.DeviceName == "" {
		return "", fmt.Errorf("no device specified")
	}

	// Prepare message
	var msg midi.Message
	channel := uint8(data.Channel - 1) // 0-based
	if channel > 15 {
		channel = 0
	}

	switch data.MsgType {
	case "note_on":
		msg = midi.NoteOn(channel, uint8(data.Note), uint8(data.Velocity))
	case "note_off":
		msg = midi.NoteOff(channel, uint8(data.Note))
	case "cc":
		msg = midi.ControlChange(channel, uint8(data.Note), uint8(data.Velocity)) // reusing Note/Velocity fields for generic Number/Value
	case "pc":
		msg = midi.ProgramChange(channel, uint8(data.Program))
	case "sysex":
		// Parse hex string
		// TODO: Implement parsing of hex string to bytes
		// For now simple placeholder
		return "", fmt.Errorf("sysex not fully implemented yet")
	default:
		return "", fmt.Errorf("unknown message type: %s", data.MsgType)
	}

	// Send message
	// we need a SendTo equivalent. midi.Manager has GetOutPort.
	outPort, err := h.midiManager.GetOutPort(data.DeviceName)
	if err != nil {
		return "", fmt.Errorf("failed to get port '%s': %v", data.DeviceName, err)
	}
	if outPort == nil {
		return "", fmt.Errorf("port '%s' not found", data.DeviceName)
	}

	send, err := midi.SendTo(outPort)
	if err != nil {
		return "", fmt.Errorf("failed to create sender: %v", err)
	}

	if err := send(msg); err != nil {
		return "", fmt.Errorf("send failed: %v", err)
	}

	return fmt.Sprintf("Sent %s to %s", data.MsgType, data.DeviceName), nil
}

func (h *MidiHandler) Validate(code string) error {
	var data MidiActionData
	if err := json.Unmarshal([]byte(code), &data); err != nil {
		return fmt.Errorf("invalid MIDI data format")
	}
	if data.DeviceName == "" {
		return fmt.Errorf("device required")
	}
	return nil
}
