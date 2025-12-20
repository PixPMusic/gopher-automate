package midi

import (
	"fmt"

	"gitlab.com/gomidi/midi/v2"
)

// ColorfulDevice implements Device for Launchpad Mini Mk3
type ColorfulDevice struct{}

func (d *ColorfulDevice) ActivateProgrammerMode(send func(midi.Message) error) error {
	// SysEx for programmer mode: 00 20 29 02 0D 0E 01
	sysexContent := []byte{0x00, 0x20, 0x29, 0x02, 0x0D, 0x0E, 0x01}
	if err := send(midi.SysEx(sysexContent)); err != nil {
		return fmt.Errorf("failed to send programmer mode message: %w", err)
	}
	return nil
}

func (d *ColorfulDevice) SetPadColor(send func(midi.Message) error, row, col int, color PadColor) error {
	// Launchpad Mini Mk3 programmer mode layout:
	// LED indices: bottom-left is 11, top-right is 99
	// Row formula: LED = (9 - row) * 10 + (col + 1)
	// Top row (row 0): 91-99
	// Bottom row (row 8): 11-19
	ledIndex := uint8((8-row)*10 + col + 11)

	// Apply gamma scaling to make colors more distinct
	r := d.scaleColor(color.R)
	g := d.scaleColor(color.G)
	b := d.scaleColor(color.B)

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

func (d *ColorfulDevice) scaleColor(value uint8) uint8 {
	if value == 0 {
		return 0
	}
	// Apply a slight curve to enhance color distinction
	f := float64(value) / 127.0
	// Use a power curve to enhance saturation
	scaled := f * f * 127.0
	if scaled < 1 && value > 0 {
		scaled = 1 // Ensure non-zero input gives non-zero output
	}
	return uint8(scaled)
}

func (d *ColorfulDevice) ClearAllPads(send func(midi.Message) error) error {
	// Send SysEx to clear all LEDs
	// Using static color 0 for all pads
	sysexContent := []byte{0x00, 0x20, 0x29, 0x02, 0x0D, 0x03}
	for i := 11; i <= 99; i++ {
		if i%10 >= 1 && i%10 <= 9 { // Valid LED indices
			sysexContent = append(sysexContent, 0x00, uint8(i), 0x00) // Static off
		}
	}
	return send(midi.SysEx(sysexContent))
}

func (d *ColorfulDevice) HandleMessage(msg midi.Message) (row, col int, isNoteOn bool, handled bool) {
	var channel, key, velocity uint8

	switch {
	case msg.GetNoteOn(&channel, &key, &velocity):
		isNoteOn = velocity > 0
		row, col = d.noteToGrid(key)
		if row >= 0 && col >= 0 {
			return row, col, isNoteOn, true
		}

	case msg.GetNoteOff(&channel, &key, &velocity):
		row, col = d.noteToGrid(key)
		if row >= 0 && col >= 0 {
			return row, col, false, true
		}

	case msg.GetControlChange(&channel, &key, &velocity):
		// Handle CC for top row buttons on Launchpad Mini Mk3 (91-98)
		if key >= 91 && key <= 98 {
			col = int(key - 91)
			isNoteOn = velocity > 0
			return 0, col, isNoteOn, true
		} else if key%10 == 9 && key >= 19 && key <= 89 {
			// Handle CC for right column buttons (19, 29... 89)
			// 19 is bottom right (Row 8), 89 is top right (Row 1)
			row = 8 - int((key-19)/10)
			isNoteOn = velocity > 0
			return row, 8, isNoteOn, true
		}
	}

	return 0, 0, false, false
}

func (d *ColorfulDevice) noteToGrid(note uint8) (int, int) {
	// Mini Mk3: LED index = (8-row)*10 + col + 11
	// Invert: row = 8 - (note-11)/10, col = (note-11)%10
	if note >= 11 && note <= 99 {
		row := 8 - int((note-11)/10)
		col := int((note - 11) % 10)
		if row >= 0 && row <= 8 && col >= 0 && col <= 8 {
			return row, col
		}
	}
	return -1, -1
}
