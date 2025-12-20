package midi

import (
	"fmt"

	"gitlab.com/gomidi/midi/v2"
)

// ClassicDevice implements Device for Launchpad S
type ClassicDevice struct{}

func (d *ClassicDevice) ActivateProgrammerMode(send func(midi.Message) error) error {
	// Launchpad S - reset to default state
	// Send reset: B0 00 00 (CC 0 value 0)
	if err := send(midi.ControlChange(0, 0, 0)); err != nil {
		return fmt.Errorf("failed to reset Launchpad S: %w", err)
	}
	return nil
}

func (d *ClassicDevice) SetPadColor(send func(midi.Message) error, row, col int, color PadColor) error {
	// Launchpad S mapping logic
	var mapping PadMapping

	// Position mapping logic (formerly GetPadMapping)
	if row == 0 && col == 8 {
		mapping = PadMapping{Exists: false}
	} else if row == 0 {
		// Top row: Control Change 104 + col
		mapping = PadMapping{IsCC: true, Number: uint8(104 + col), Exists: true}
	} else {
		// Grid and right column: Note messages
		// Row 1 = notes 0-8, Row 2 = notes 16-24, etc.
		noteNum := uint8((row-1)*16 + col)
		mapping = PadMapping{IsCC: false, Number: noteNum, Exists: true}
	}

	if !mapping.Exists {
		return nil
	}

	// Velocity calculation logic (formerly setColorLaunchpadS)
	var velocity uint8

	if color.R < 5 && color.G < 5 && color.B < 5 {
		velocity = 0x0C // flags only, no color = off
	} else {
		effectiveR := int(color.R) + int(color.B)/4
		effectiveG := int(color.G) + (int(color.B)*3)/4

		if effectiveR > 127 {
			effectiveR = 127
		}
		if effectiveG > 127 {
			effectiveG = 127
		}

		redLevel := d.colorTo4Level(uint8(effectiveR))
		greenLevel := d.colorTo4Level(uint8(effectiveG))

		velocity = (greenLevel << 4) | 0x0C | redLevel
	}

	if mapping.IsCC {
		return send(midi.ControlChange(0, mapping.Number, velocity))
	}
	return send(midi.NoteOn(0, mapping.Number, velocity))
}

func (d *ClassicDevice) colorTo4Level(value uint8) uint8 {
	if value < 32 {
		return 0
	} else if value < 64 {
		return 1
	} else if value < 96 {
		return 2
	}
	return 3
}

func (d *ClassicDevice) ClearAllPads(send func(midi.Message) error) error {
	// Reset Launchpad S: B0 00 00
	return send(midi.ControlChange(0, 0, 0))
}

func (d *ClassicDevice) HandleMessage(msg midi.Message) (row, col int, isNoteOn bool, handled bool) {
	var channel, key, velocity uint8

	switch {
	case msg.GetNoteOn(&channel, &key, &velocity):
		isNoteOn = velocity > 0
		// Launchpad S layout: Row 1 = notes 0-8, Row 2 = notes 16-24, etc.
		row = int(key/16) + 1
		col = int(key % 16)
		if row >= 1 && row <= 8 && col >= 0 && col <= 8 {
			return row, col, isNoteOn, true
		}

	case msg.GetNoteOff(&channel, &key, &velocity):
		// Launchpad S layout
		row = int(key/16) + 1
		col = int(key % 16)
		if row >= 1 && row <= 8 && col >= 0 && col <= 8 {
			return row, col, false, true
		}

	case msg.GetControlChange(&channel, &key, &velocity):
		// Handle CC for top row buttons on Launchpad S (104-111)
		if key >= 104 && key <= 111 {
			col = int(key - 104)
			isNoteOn = velocity > 0
			return 0, col, isNoteOn, true
		}
	}

	return 0, 0, false, false
}
