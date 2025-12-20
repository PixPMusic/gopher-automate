package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/PixPMusic/gopher-automate/internal/actions"
	"github.com/google/uuid"
)

// DeviceType represents the type of MIDI device
type DeviceType string

const (
	DeviceTypeClassic  DeviceType = "classic"  // Launchpad S
	DeviceTypeColorful DeviceType = "colorful" // Launchpad Mini Mk3
	DeviceTypeGeneric  DeviceType = "generic"  // Generic MIDI for inter-app communication
)

// PadColorConfig stores RGB colors for a pad (all values 0-127)
type PadColorConfig struct {
	// Button color (full RGB for colorful devices)
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`

	// Classic color (for Launchpad S devices, red/green only)
	ClassicR uint8 `json:"classic_r"`
	ClassicG uint8 `json:"classic_g"`
	ClassicB uint8 `json:"classic_b"`

	// Pressed color (shown when pad is pressed)
	PressedR uint8 `json:"pressed_r"`
	PressedG uint8 `json:"pressed_g"`
	PressedB uint8 `json:"pressed_b"`

	// Classic pressed color
	ClassicPressedR uint8 `json:"classic_pressed_r"`
	ClassicPressedG uint8 `json:"classic_pressed_g"`
	ClassicPressedB uint8 `json:"classic_pressed_b"`

	// Link flags - when true, classic color is auto-derived from button color
	LinkButtonClassic  bool `json:"link_button_classic"`
	LinkPressedClassic bool `json:"link_pressed_classic"`

	// ActionID is the ID of the action to execute when this pad is pressed
	ActionID string `json:"action_id,omitempty"`
}

// CalculateClassicColor converts full RGB to the classic device's approximation.
// Returns the RGB values that would be displayed on a classic (red/green only) device.
// This uses the same algorithm as the MIDI package for consistency.
func CalculateClassicColor(r, g, b uint8) (uint8, uint8, uint8) {
	// Blue leans toward green since that's closer on the spectrum
	effectiveR := int(r) + int(b)/4
	effectiveG := int(g) + (int(b)*3)/4

	if effectiveR > 127 {
		effectiveR = 127
	}
	if effectiveG > 127 {
		effectiveG = 127
	}

	// Map to 4 intensity levels (0-3) and back to preview values
	redLevel := colorTo4Level(uint8(effectiveR))
	greenLevel := colorTo4Level(uint8(effectiveG))

	// Convert back to 0-127 for preview (0->0, 1->42, 2->85, 3->127)
	previewR := redLevel * 42
	previewG := greenLevel * 42
	if redLevel == 3 {
		previewR = 127
	}
	if greenLevel == 3 {
		previewG = 127
	}

	return previewR, previewG, 0 // Classic has no blue
}

// CalculateClassicLevel converts full RGB to 0-3 intensity levels for classic device.
// Returns (redLevel, greenLevel) where each is 0-3.
func CalculateClassicLevel(r, g, b uint8) (uint8, uint8) {
	effectiveR := int(r) + int(b)/4
	effectiveG := int(g) + (int(b)*3)/4

	if effectiveR > 127 {
		effectiveR = 127
	}
	if effectiveG > 127 {
		effectiveG = 127
	}

	return colorTo4Level(uint8(effectiveR)), colorTo4Level(uint8(effectiveG))
}

// LevelTo127 converts a 0-3 level to 0-127 value for storage/MIDI
func LevelTo127(level uint8) uint8 {
	switch level {
	case 0:
		return 0
	case 1:
		return 42
	case 2:
		return 85
	default:
		return 127
	}
}

// Level127To4 converts a 0-127 value back to 0-3 level for UI sliders
func Level127To4(value uint8) uint8 {
	if value < 21 {
		return 0
	} else if value < 64 {
		return 1
	} else if value < 106 {
		return 2
	}
	return 3
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

// MenuLayout stores the 9x9 grid of pad colors
type MenuLayout struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Colors       [9][9]PadColorConfig `json:"colors"`        // [row][col] - Standard 9x9 (including top row/right col)
	LeftColors   [8]PadColorConfig    `json:"left_colors"`   // Pro: Left column (Rows 1-8)
	BottomColors [8]PadColorConfig    `json:"bottom_colors"` // Pro: Bottom row (Cols 1-8)

	// Pro+ (MK3) Additions
	TopLeftColor         PadColorConfig    `json:"top_left_color"`         // (0,0)
	BottomLeftColor      PadColorConfig    `json:"bottom_left_color"`      // (9,0)
	BottomRightColor     PadColorConfig    `json:"bottom_right_color"`     // (9,9)
	ExtendedBottomColors [8]PadColorConfig `json:"extended_bottom_colors"` // Row 10 (Cols 1-8)
}

// NewMenuLayout creates a new menu layout with all pads off
func NewMenuLayout() MenuLayout {
	layout := MenuLayout{
		ID:   uuid.New().String(),
		Name: "Main Menu",
	}
	// All colors default to zero (off)
	return layout
}

// DeviceConfig holds configuration for a single MIDI device
type DeviceConfig struct {
	ID       string     `json:"id"`        // Unique identifier
	Name     string     `json:"name"`      // User-friendly name
	InPort   string     `json:"in_port"`   // MIDI input port name
	OutPort  string     `json:"out_port"`  // MIDI output port name
	Type     DeviceType `json:"type"`      // Classic or Colorful
	MainMenu string     `json:"main_menu"` // Menu assignment (placeholder)
}

// NewDeviceConfig creates a new device config with a generated ID
func NewDeviceConfig() DeviceConfig {
	return DeviceConfig{
		ID:   uuid.New().String(),
		Name: "New Device",
		Type: DeviceTypeClassic,
	}
}

// MessageMapping maps a MIDI message to an action for inter-app communication
type MessageMapping struct {
	ID          string `json:"id"`
	Name        string `json:"name"`         // User-friendly description
	MessageType string `json:"message_type"` // "note", "cc", "program_change"
	Channel     int    `json:"channel"`      // 0-15, or -1 for any channel
	Number      int    `json:"number"`       // Note/CC number (0-127)
	ActionID    string `json:"action_id"`    // Action to trigger
}

// NewMessageMapping creates a new message mapping with a generated ID
func NewMessageMapping() MessageMapping {
	return MessageMapping{
		ID:          uuid.New().String(),
		Name:        "New Mapping",
		MessageType: "note",
		Channel:     -1,
		Number:      60,
	}
}

// Config holds application configuration
type Config struct {
	FirstLaunchCompleted   bool                  `json:"first_launch_completed"`
	OpenAtStartup          bool                  `json:"open_at_startup"`
	SuppressUnsavedWarning bool                  `json:"suppress_unsaved_warning"`
	Devices                []DeviceConfig        `json:"devices"`
	Menus                  []MenuLayout          `json:"menus"`
	CurrentMenuID          string                `json:"current_menu_id"`
	Actions                []actions.Action      `json:"actions"`
	ActionGroups           []actions.ActionGroup `json:"action_groups"`
	MessageMappings        []MessageMapping      `json:"message_mappings"`
}

// configDir returns the platform-appropriate config directory
func configDir() (string, error) {
	configHome, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configHome, "gopher-automate"), nil
}

// ConfigPath returns the full path to the config file
func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from disk, returning defaults if not found
func Load() (*Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		// Return default config with one default menu
		defaultMenu := NewMenuLayout()
		return &Config{
			FirstLaunchCompleted: false,
			OpenAtStartup:        false,
			Devices:              []DeviceConfig{},
			Menus:                []MenuLayout{defaultMenu},
			CurrentMenuID:        defaultMenu.ID,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Ensure slices are not nil
	if cfg.Devices == nil {
		cfg.Devices = []DeviceConfig{}
	}
	if cfg.Menus == nil || len(cfg.Menus) == 0 {
		defaultMenu := NewMenuLayout()
		cfg.Menus = []MenuLayout{defaultMenu}
		cfg.CurrentMenuID = defaultMenu.ID
	}

	return &cfg, nil
}

// Save writes the config to disk
func (c *Config) Save() error {
	configPath, err := ConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetCurrentMenu returns the current menu layout
func (c *Config) GetCurrentMenu() *MenuLayout {
	for i := range c.Menus {
		if c.Menus[i].ID == c.CurrentMenuID {
			return &c.Menus[i]
		}
	}
	if len(c.Menus) > 0 {
		return &c.Menus[0]
	}
	return nil
}

// AddDevice adds a new device to the config
func (c *Config) AddDevice(device DeviceConfig) {
	c.Devices = append(c.Devices, device)
}

// RemoveDevice removes a device by ID
func (c *Config) RemoveDevice(id string) {
	for i, d := range c.Devices {
		if d.ID == id {
			c.Devices = append(c.Devices[:i], c.Devices[i+1:]...)
			return
		}
	}
}

// UpdateDevice updates an existing device by ID
func (c *Config) UpdateDevice(device DeviceConfig) {
	for i, d := range c.Devices {
		if d.ID == device.ID {
			c.Devices[i] = device
			return
		}
	}
}

// GetActionStore returns an ActionStore populated with config's actions and groups
func (c *Config) GetActionStore() *actions.ActionStore {
	store := actions.NewActionStore()
	store.Actions = c.Actions
	store.Groups = c.ActionGroups
	if store.Actions == nil {
		store.Actions = []actions.Action{}
	}
	if store.Groups == nil {
		store.Groups = []actions.ActionGroup{}
	}
	return store
}

// SyncActionStore updates the config's actions and groups from an ActionStore
func (c *Config) SyncActionStore(store *actions.ActionStore) {
	c.Actions = store.Actions
	c.ActionGroups = store.Groups
}

// GetAction returns an action by ID, or nil if not found
func (c *Config) GetAction(id string) *actions.Action {
	for i := range c.Actions {
		if c.Actions[i].ID == id {
			return &c.Actions[i]
		}
	}
	return nil
}
