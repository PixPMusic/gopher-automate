package window

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/PixPMusic/gopher-automate/internal/actions"
	"github.com/PixPMusic/gopher-automate/internal/config"
	"github.com/PixPMusic/gopher-automate/internal/midi"
)

// MainWindow manages the main application window
type MainWindow struct {
	window      fyne.Window
	app         fyne.App
	cfg         *config.Config
	midiManager *midi.Manager
	deviceList  *widget.List
	onSave      func()

	// Menu editor state
	gridRects      [9][9]*canvas.Rectangle
	layoutDropdown *widget.Select
	gridContainer  *fyne.Container
	revertBtn      *widget.Button
	dirty          bool // true if current layout has unsaved changes

	// Color picker panel state
	selectedRow int
	selectedCol int
	colorPanel  *fyne.Container

	// Color sliders (0-127 range)
	buttonRSlider, buttonGSlider, buttonBSlider                         *widget.Slider
	classicRSlider, classicGSlider, classicBSlider                      *widget.Slider
	pressedRSlider, pressedGSlider, pressedBSlider                      *widget.Slider
	classicPressedRSlider, classicPressedGSlider, classicPressedBSlider *widget.Slider

	// Color previews
	buttonPreview, classicPreview, pressedPreview, classicPressedPreview *canvas.Rectangle

	// Link checkboxes
	linkButtonClassic, linkPressedClassic *widget.Check

	// MIDI input listeners
	midiStopFuncs []func()

	// Action system
	executor         *actions.Executor
	actionStore      *actions.ActionStore
	actionList       *widget.List
	actionEditor     *fyne.Container
	selectedAction   *actions.Action
	selectedGroup    *actions.ActionGroup
	actionNameEntry  *widget.Entry
	actionTypeSelect *widget.Select
	actionCodeEntry  *widget.Entry
	actionFeedback   *widget.Label
	padActionSelect  *widget.Select // Action selector in color picker panel

	// Specialized editor fields
	sleepDurationEntry     *widget.Entry
	waitForCompletionCheck *widget.Check

	// MIDI Action Editor fields
	midiDeviceSelect  *widget.Select
	midiMsgTypeSelect *widget.RadioGroup
	midiChannelSelect *widget.Select
	midiNoteEntry     *widget.Entry
	midiVelocityEntry *widget.Entry
	midiProgramEntry  *widget.Entry
	midiSysexEntry    *widget.Entry

	actionEditorContent *fyne.Container // Container for swapping editor content

	syntaxHighlighter *SyntaxHighlighter
	codePreviewScroll *container.Scroll

	// Message Mapping system
	mappingList *widget.List
}

// NewMainWindow creates the main application window
func NewMainWindow(app fyne.App, cfg *config.Config, midiManager *midi.Manager, onSave func()) *MainWindow {
	win := app.NewWindow("GopherAutomate")

	mw := &MainWindow{
		window:            win,
		app:               app,
		cfg:               cfg,
		midiManager:       midiManager,
		onSave:            onSave,
		executor:          actions.NewExecutor(midiManager),
		actionStore:       cfg.GetActionStore(),
		syntaxHighlighter: NewSyntaxHighlighter(),
	}

	mw.setupUI()

	win.Resize(fyne.NewSize(950, 660))
	win.CenterOnScreen()

	win.SetCloseIntercept(func() {
		win.Hide()
	})

	return mw
}

// InitializeDevices puts all devices in programmer mode and sends current layout
func (mw *MainWindow) InitializeDevices() {
	for _, device := range mw.cfg.Devices {
		if device.OutPort == "" {
			continue
		}
		deviceType := midi.DeviceType(device.Type)
		if err := mw.midiManager.ActivateProgrammerMode(device.OutPort, deviceType); err != nil {
			log.Printf("Failed to activate programmer mode for %s: %v", device.Name, err)
		} else {
			log.Printf("Activated programmer mode for %s", device.Name)
		}
	}
	// Send current layout to all devices
	mw.sendGridToDevices()

	// Start MIDI input listeners
	mw.StartMIDIListeners()
}

// StartMIDIListeners begins listening for MIDI input from all configured devices
func (mw *MainWindow) StartMIDIListeners() {
	mw.StopMIDIListeners() // Stop any existing listeners

	for _, device := range mw.cfg.Devices {
		if device.InPort == "" {
			continue
		}

		deviceType := midi.DeviceType(device.Type)
		menuName := device.MainMenu

		var stop func()
		var err error

		if device.Type == config.DeviceTypeGeneric {
			// Generic devices use message mapping instead of pad layout
			stop, err = mw.midiManager.StartGenericListening(device.InPort, func(portName, msgType string, channel, number, value int) {
				mw.handleGenericMIDIMessage(portName, msgType, channel, number, value)
			})
		} else {
			// Launchpad devices use pad layout
			stop, err = mw.midiManager.StartListening(device.InPort, deviceType, func(portName string, row, col int, isNoteOn bool) {
				mw.handlePadPress(menuName, row, col, isNoteOn)
			})
		}

		if err != nil {
			log.Printf("Failed to start listener for %s: %v", device.Name, err)
			continue
		}

		if stop != nil {
			mw.midiStopFuncs = append(mw.midiStopFuncs, stop)
			log.Printf("Started listening on %s", device.InPort)
		}
	}
}

// StopMIDIListeners stops all MIDI input listeners
func (mw *MainWindow) StopMIDIListeners() {
	for _, stop := range mw.midiStopFuncs {
		if stop != nil {
			stop()
		}
	}
	mw.midiStopFuncs = nil
}

// handlePadPress sends pressed/unpressed color to all devices with the same menu
func (mw *MainWindow) handlePadPress(menuName string, row, col int, isNoteOn bool) {
	if menuName == "" {
		return
	}

	// Find the menu
	var menu *config.MenuLayout
	for i := range mw.cfg.Menus {
		if mw.cfg.Menus[i].Name == menuName {
			menu = &mw.cfg.Menus[i]
			break
		}
	}
	if menu == nil {
		return
	}

	padColor := menu.Colors[row][col]

	// Execute assigned action on Note On (pad pressed)
	// Execute assigned action on Note On (pad pressed)
	if isNoteOn && padColor.ActionID != "" {
		mw.resolveAndRun(padColor.ActionID)
	}

	// Send to all devices with this menu
	for _, device := range mw.cfg.Devices {
		if device.OutPort == "" || device.MainMenu != menuName {
			continue
		}

		deviceType := midi.DeviceType(device.Type)
		var midiColor midi.PadColor

		if isNoteOn {
			// Use pressed color
			if deviceType == midi.DeviceTypeClassic {
				midiColor = midi.PadColor{R: padColor.ClassicPressedR, G: padColor.ClassicPressedG, B: padColor.ClassicPressedB}
			} else {
				midiColor = midi.PadColor{R: padColor.PressedR, G: padColor.PressedG, B: padColor.PressedB}
			}
		} else {
			// Restore button color
			if deviceType == midi.DeviceTypeClassic {
				midiColor = midi.PadColor{R: padColor.ClassicR, G: padColor.ClassicG, B: padColor.ClassicB}
			} else {
				midiColor = midi.PadColor{R: padColor.R, G: padColor.G, B: padColor.B}
			}
		}

		if err := mw.midiManager.SetPadColor(device.OutPort, deviceType, row, col, midiColor); err != nil {
			log.Printf("Failed to set pad color: %v", err)
		}
	}
}

// handleGenericMIDIMessage handles MIDI messages from Generic devices for inter-app communication
func (mw *MainWindow) handleGenericMIDIMessage(portName, msgType string, channel, number, value int) {
	// Only trigger on "on" events (velocity/value > 0)
	if value == 0 {
		return
	}

	// Find matching message mappings
	for _, mapping := range mw.cfg.MessageMappings {
		if mw.mappingMatches(mapping, msgType, channel, number) {
			mw.resolveAndRun(mapping.ActionID)
		}
	}
}

// mappingMatches checks if a MIDI message matches a mapping
func (mw *MainWindow) mappingMatches(mapping config.MessageMapping, msgType string, channel, number int) bool {
	// Check message type
	if mapping.MessageType != msgType {
		return false
	}

	// Check channel (-1 means any channel)
	if mapping.Channel != -1 && mapping.Channel != channel {
		return false
	}

	// Check number
	if mapping.Number != number {
		return false
	}

	return true
}

func (mw *MainWindow) setupUI() {
	devicesTab := container.NewTabItem("Devices", mw.createDevicesTab())
	menuEditorTab := container.NewTabItem("Menu Editor", mw.createMenuEditorTab())
	actionsTab := container.NewTabItem("Actions", mw.createActionsTab())
	messageMappingTab := container.NewTabItem("Message Mapping", mw.createMessageMappingTab())

	tabs := container.NewAppTabs(devicesTab, menuEditorTab, actionsTab, messageMappingTab)
	tabs.SetTabLocation(container.TabLocationTop)

	mw.window.SetContent(tabs)
}

// Show displays the window
func (mw *MainWindow) Show() {
	mw.deviceList.Refresh()
	mw.layoutDropdown.Options = mw.getLayoutNames()
	mw.layoutDropdown.SetSelected(mw.getCurrentLayoutName())
	mw.window.Show()
}

// Hide hides the window
func (mw *MainWindow) Hide() {
	mw.window.Hide()
}

// Window returns the underlying fyne.Window
func (mw *MainWindow) Window() fyne.Window {
	return mw.window
}
