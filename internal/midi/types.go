package midi

// DeviceType represents the type of device
type DeviceType string

const (
	DeviceTypeClassic  DeviceType = "classic"  // Launchpad S - no special programmer mode
	DeviceTypeColorful DeviceType = "colorful" // Launchpad Mini Mk3 - requires SysEx
	DeviceTypeGeneric  DeviceType = "generic"  // Generic MIDI for inter-app communication
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
