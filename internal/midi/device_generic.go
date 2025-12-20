package midi

import "gitlab.com/gomidi/midi/v2"

// GenericDevice implements Device for Generic MIDI interaction
// It basically does nothing for grid operations as it uses message mapping.
type GenericDevice struct{}

func (d *GenericDevice) ActivateProgrammerMode(send func(midi.Message) error) error {
	return nil
}

func (d *GenericDevice) SetPadColor(send func(midi.Message) error, row, col int, color PadColor) error {
	return nil
}

func (d *GenericDevice) ClearAllPads(send func(midi.Message) error) error {
	return nil
}

func (d *GenericDevice) HandleMessage(msg midi.Message) (row, col int, isNoteOn bool, handled bool) {
	return 0, 0, false, false
}
