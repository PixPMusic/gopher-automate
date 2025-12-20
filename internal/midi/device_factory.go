package midi

// GetDevice returns the appropriate Device implementation for the given type
func GetDevice(deviceType DeviceType) Device {
	switch deviceType {
	case DeviceTypeClassic:
		return &ClassicDevice{}
	case DeviceTypeColorful:
		return &ColorfulDevice{}
	case DeviceTypeGeneric:
		return &GenericDevice{}
	default:
		// Default to Colorful as fallback (matching previous behavior)
		return &ColorfulDevice{}
	}
}
