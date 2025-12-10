package window

import (
	"image"
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/PixPMusic/gopher-automate/internal/config"
	"github.com/PixPMusic/gopher-automate/internal/midi"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
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
}

// NewMainWindow creates the main application window
func NewMainWindow(app fyne.App, cfg *config.Config, midiManager *midi.Manager, onSave func()) *MainWindow {
	win := app.NewWindow("GopherAutomate")

	mw := &MainWindow{
		window:      win,
		app:         app,
		cfg:         cfg,
		midiManager: midiManager,
		onSave:      onSave,
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

		stop, err := mw.midiManager.StartListening(device.InPort, deviceType, func(portName string, row, col int, isNoteOn bool) {
			mw.handlePadPress(menuName, row, col, isNoteOn)
		})

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

func (mw *MainWindow) setupUI() {
	devicesTab := container.NewTabItem("Devices", mw.createDevicesTab())
	menuEditorTab := container.NewTabItem("Menu Editor", mw.createMenuEditorTab())

	tabs := container.NewAppTabs(devicesTab, menuEditorTab)
	tabs.SetTabLocation(container.TabLocationTop)

	mw.window.SetContent(tabs)
}

// ============ DEVICES TAB ============

func (mw *MainWindow) createDevicesTab() fyne.CanvasObject {
	devicesHeader := widget.NewLabel("MIDI Devices")
	devicesHeader.TextStyle = fyne.TextStyle{Bold: true}

	addBtn := widget.NewButtonWithIcon("Add Device", theme.ContentAddIcon(), func() {
		mw.addDevice()
	})

	devicesToolbar := container.NewBorder(nil, nil, devicesHeader, addBtn)

	headerName := widget.NewLabel("Name")
	headerName.TextStyle = fyne.TextStyle{Bold: true}
	headerIn := widget.NewLabel("Input Port")
	headerIn.TextStyle = fyne.TextStyle{Bold: true}
	headerOut := widget.NewLabel("Output Port")
	headerOut.TextStyle = fyne.TextStyle{Bold: true}
	headerType := widget.NewLabel("Type")
	headerType.TextStyle = fyne.TextStyle{Bold: true}
	headerMenu := widget.NewLabel("Menu")
	headerMenu.TextStyle = fyne.TextStyle{Bold: true}
	headerActions := widget.NewLabel("")

	columnHeaders := container.NewGridWithColumns(6,
		headerName, headerIn, headerOut, headerType, headerMenu, headerActions,
	)

	mw.deviceList = widget.NewList(
		func() int { return len(mw.cfg.Devices) },
		func() fyne.CanvasObject { return mw.createDeviceRow() },
		func(id widget.ListItemID, obj fyne.CanvasObject) { mw.updateDeviceRow(id, obj) },
	)

	saveBtn := widget.NewButtonWithIcon("Save & Activate Devices", theme.DocumentSaveIcon(), func() {
		mw.saveAndActivate()
	})
	saveBtn.Importance = widget.HighImportance

	actionsSection := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(saveBtn),
	)

	return container.NewBorder(
		container.NewVBox(devicesToolbar, widget.NewSeparator(), columnHeaders),
		actionsSection,
		nil, nil,
		mw.deviceList,
	)
}

func (mw *MainWindow) createDeviceRow() fyne.CanvasObject {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Device Name")

	inPortSelect := widget.NewSelect([]string{}, nil)
	inPortSelect.PlaceHolder = "Select..."

	outPortSelect := widget.NewSelect([]string{}, nil)
	outPortSelect.PlaceHolder = "Select..."

	typeSelect := widget.NewSelect([]string{"Classic", "Colorful"}, nil)
	typeSelect.PlaceHolder = "Type"

	menuSelect := widget.NewSelect([]string{"(None)"}, nil)
	menuSelect.PlaceHolder = "Menu"

	removeBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)

	return container.NewGridWithColumns(6,
		nameEntry, inPortSelect, outPortSelect, typeSelect, menuSelect,
		container.NewCenter(removeBtn),
	)
}

func (mw *MainWindow) updateDeviceRow(id widget.ListItemID, obj fyne.CanvasObject) {
	if id >= len(mw.cfg.Devices) {
		return
	}

	device := &mw.cfg.Devices[id]
	grid := obj.(*fyne.Container)

	nameEntry := grid.Objects[0].(*widget.Entry)
	inPortSelect := grid.Objects[1].(*widget.Select)
	outPortSelect := grid.Objects[2].(*widget.Select)
	typeSelect := grid.Objects[3].(*widget.Select)
	menuSelect := grid.Objects[4].(*widget.Select)
	removeBtnContainer := grid.Objects[5].(*fyne.Container)
	removeBtn := removeBtnContainer.Objects[0].(*widget.Button)

	inPorts := mw.midiManager.ListInPorts()
	outPorts := mw.midiManager.ListOutPorts()

	inPortSelect.Options = append([]string{"(None)"}, inPorts...)
	outPortSelect.Options = append([]string{"(None)"}, outPorts...)

	nameEntry.SetText(device.Name)
	nameEntry.OnChanged = func(s string) { device.Name = s }

	if device.InPort == "" {
		inPortSelect.SetSelected("(None)")
	} else {
		inPortSelect.SetSelected(device.InPort)
	}
	inPortSelect.OnChanged = func(s string) {
		if s == "(None)" {
			device.InPort = ""
		} else {
			device.InPort = s
		}
	}

	if device.OutPort == "" {
		outPortSelect.SetSelected("(None)")
	} else {
		outPortSelect.SetSelected(device.OutPort)
	}
	outPortSelect.OnChanged = func(s string) {
		if s == "(None)" {
			device.OutPort = ""
		} else {
			device.OutPort = s
		}
	}

	switch device.Type {
	case config.DeviceTypeClassic:
		typeSelect.SetSelected("Classic")
	case config.DeviceTypeColorful:
		typeSelect.SetSelected("Colorful")
	default:
		typeSelect.SetSelected("Classic")
	}
	typeSelect.OnChanged = func(s string) {
		switch s {
		case "Classic":
			device.Type = config.DeviceTypeClassic
		case "Colorful":
			device.Type = config.DeviceTypeColorful
		}
	}

	// Populate menu dropdown with available layouts
	menuOptions := []string{"(None)"}
	for _, m := range mw.cfg.Menus {
		menuOptions = append(menuOptions, m.Name)
	}
	menuSelect.Options = menuOptions
	if device.MainMenu == "" {
		menuSelect.SetSelected("(None)")
	} else {
		menuSelect.SetSelected(device.MainMenu)
	}
	menuSelect.OnChanged = func(s string) {
		if s == "(None)" {
			device.MainMenu = ""
		} else {
			device.MainMenu = s
		}
	}

	deviceID := device.ID
	removeBtn.OnTapped = func() { mw.removeDevice(deviceID) }
}

func (mw *MainWindow) addDevice() {
	newDevice := config.NewDeviceConfig()
	mw.cfg.AddDevice(newDevice)
	mw.deviceList.Refresh()
}

func (mw *MainWindow) removeDevice(id string) {
	mw.cfg.RemoveDevice(id)
	mw.deviceList.Refresh()
}

func (mw *MainWindow) saveAndActivate() {
	// Reload menus from disk to avoid saving unsaved layout changes
	if savedCfg, err := config.Load(); err == nil {
		mw.cfg.Menus = savedCfg.Menus
	}

	if err := mw.cfg.Save(); err != nil {
		log.Printf("Failed to save config: %v", err)
		return
	}

	// Initialize devices and send layout
	mw.InitializeDevices()

	// Also refresh the grid to show saved state
	mw.setDirty(false)
	mw.refreshGrid()

	if mw.onSave != nil {
		mw.onSave()
	}
}

// ============ MENU EDITOR TAB ============

func (mw *MainWindow) createMenuEditorTab() fyne.CanvasObject {
	header := widget.NewLabel("Menu Editor")
	header.TextStyle = fyne.TextStyle{Bold: true}

	// Create grid container FIRST (before dropdown can trigger refresh)
	mw.gridContainer = container.NewCenter(mw.createPadGrid())

	// Layout dropdown - create without callback initially
	layoutLabel := widget.NewLabel("Layout:")
	mw.layoutDropdown = widget.NewSelect(mw.getLayoutNames(), nil)
	mw.layoutDropdown.SetSelected(mw.getCurrentLayoutName())
	// Now set the callback after initial selection
	mw.layoutDropdown.OnChanged = func(selected string) {
		mw.loadLayoutByName(selected)
	}

	// New layout button
	newBtn := widget.NewButtonWithIcon("New", theme.ContentAddIcon(), func() {
		mw.createNewLayout()
	})

	// Delete layout button
	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		mw.deleteCurrentLayout()
	})

	// Rename button
	renameBtn := widget.NewButtonWithIcon("Rename", theme.DocumentCreateIcon(), func() {
		mw.renameCurrentLayout()
	})

	layoutBar := container.NewHBox(layoutLabel, mw.layoutDropdown, newBtn, renameBtn, deleteBtn)

	subtitle := widget.NewLabel("Click a pad to select it, then adjust colors in the panel.")

	// Action buttons
	mw.revertBtn = widget.NewButtonWithIcon("Revert", theme.ContentUndoIcon(), func() {
		mw.revertLayout()
	})
	mw.revertBtn.Disable() // Start disabled since no changes yet

	clearBtn := widget.NewButtonWithIcon("Clear All", theme.ContentClearIcon(), func() {
		mw.clearGrid()
	})

	saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		mw.saveLayout()
	})
	saveBtn.Importance = widget.HighImportance

	saveAsNewBtn := widget.NewButtonWithIcon("Save As New", theme.ContentAddIcon(), func() {
		mw.saveAsNewLayout()
	})

	actions := container.NewHBox(mw.revertBtn, clearBtn, saveBtn, saveAsNewBtn)

	// Create color picker panel on right
	mw.colorPanel = mw.createColorPickerPanel()

	// Horizontal split: grid on left, color picker on right
	split := container.NewHSplit(mw.gridContainer, container.NewVScroll(mw.colorPanel))
	split.Offset = 0.50

	return container.NewBorder(
		container.NewVBox(header, layoutBar, subtitle, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), actions),
		nil, nil,
		split,
	)
}

func (mw *MainWindow) getLayoutNames() []string {
	names := make([]string, len(mw.cfg.Menus))
	for i, m := range mw.cfg.Menus {
		names[i] = m.Name
	}
	return names
}

func (mw *MainWindow) getCurrentLayoutName() string {
	menu := mw.cfg.GetCurrentMenu()
	if menu != nil {
		return menu.Name
	}
	return ""
}

func (mw *MainWindow) loadLayoutByName(name string) {
	// Skip if this is the same layout (prevents callback loops)
	if name == mw.getCurrentLayoutName() {
		return
	}

	// Check if we have unsaved changes
	if mw.dirty && !mw.cfg.SuppressUnsavedWarning {
		mw.showUnsavedWarning(name)
		return
	}
	mw.doLoadLayout(name)
}

func (mw *MainWindow) showUnsavedWarning(targetName string) {
	currentName := mw.getCurrentLayoutName()
	dontShowAgain := widget.NewCheck("Don't show this warning again", nil)

	content := container.NewVBox(
		widget.NewLabel("You have unsaved changes that will be lost."),
		widget.NewLabel("Do you want to continue?"),
		dontShowAgain,
	)

	dialog.ShowCustomConfirm("Unsaved Changes", "Continue", "Cancel", content, func(confirm bool) {
		if confirm {
			if dontShowAgain.Checked {
				mw.cfg.SuppressUnsavedWarning = true
			}
			// Reload config from disk to discard unsaved changes
			if newCfg, err := config.Load(); err == nil {
				mw.cfg.Menus = newCfg.Menus
			}
			mw.setDirty(false)
			mw.doLoadLayout(targetName)
		} else {
			// Revert dropdown to current layout without triggering callback
			mw.layoutDropdown.OnChanged = nil
			mw.layoutDropdown.SetSelected(currentName)
			mw.layoutDropdown.OnChanged = func(selected string) {
				mw.loadLayoutByName(selected)
			}
		}
	}, mw.window)
}

func (mw *MainWindow) doLoadLayout(name string) {
	for i := range mw.cfg.Menus {
		if mw.cfg.Menus[i].Name == name {
			mw.cfg.CurrentMenuID = mw.cfg.Menus[i].ID

			// Ensure legacy/uninitialized colors are linked and converted
			mw.ensureDefaultLinking(&mw.cfg.Menus[i])

			mw.setDirty(false)
			mw.refreshGrid()
			return
		}
	}
}

func (mw *MainWindow) createNewLayout() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Layout Name")
	entry.SetText("New Layout")

	dialog.ShowCustomConfirm("Create New Layout", "Create", "Cancel",
		container.NewVBox(widget.NewLabel("Enter a name for the new layout:"), entry),
		func(confirm bool) {
			if confirm && entry.Text != "" {
				newMenu := config.NewMenuLayout()
				newMenu.Name = entry.Text
				mw.cfg.Menus = append(mw.cfg.Menus, newMenu)
				mw.cfg.CurrentMenuID = newMenu.ID
				mw.layoutDropdown.Options = mw.getLayoutNames()
				mw.layoutDropdown.SetSelected(newMenu.Name)
				mw.refreshGrid()
				mw.cfg.Save()
			}
		}, mw.window)
}

func (mw *MainWindow) deleteCurrentLayout() {
	if len(mw.cfg.Menus) <= 1 {
		dialog.ShowInformation("Cannot Delete", "You must have at least one layout.", mw.window)
		return
	}

	menu := mw.cfg.GetCurrentMenu()
	if menu == nil {
		return
	}

	dialog.ShowConfirm("Delete Layout", "Are you sure you want to delete '"+menu.Name+"'?",
		func(confirm bool) {
			if confirm {
				// Find and remove the menu
				for i, m := range mw.cfg.Menus {
					if m.ID == menu.ID {
						mw.cfg.Menus = append(mw.cfg.Menus[:i], mw.cfg.Menus[i+1:]...)
						break
					}
				}
				// Switch to first available menu
				if len(mw.cfg.Menus) > 0 {
					mw.cfg.CurrentMenuID = mw.cfg.Menus[0].ID
				}
				mw.layoutDropdown.Options = mw.getLayoutNames()
				mw.layoutDropdown.SetSelected(mw.getCurrentLayoutName())
				mw.refreshGrid()
				mw.cfg.Save()
				mw.sendGridToDevices()
			}
		}, mw.window)
}

func (mw *MainWindow) renameCurrentLayout() {
	menu := mw.cfg.GetCurrentMenu()
	if menu == nil {
		return
	}

	entry := widget.NewEntry()
	entry.SetText(menu.Name)

	dialog.ShowCustomConfirm("Rename Layout", "Rename", "Cancel",
		container.NewVBox(widget.NewLabel("Enter a new name:"), entry),
		func(confirm bool) {
			if confirm && entry.Text != "" {
				menu.Name = entry.Text
				mw.layoutDropdown.Options = mw.getLayoutNames()
				mw.layoutDropdown.SetSelected(menu.Name)
				mw.cfg.Save()
			}
		}, mw.window)
}

func (mw *MainWindow) refreshGrid() {
	// Update all grid rectangles with colors from current menu
	menu := mw.cfg.GetCurrentMenu()
	if menu == nil {
		return
	}

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			c := menu.Colors[row][col]
			mw.gridRects[row][col].FillColor = color.RGBA{
				R: uint8(c.R * 2),
				G: uint8(c.G * 2),
				B: uint8(c.B * 2),
				A: 255,
			}
			mw.gridRects[row][col].Refresh()
		}
	}
}

func (mw *MainWindow) createPadGrid() fyne.CanvasObject {
	grid := container.NewGridWithColumns(9)

	menu := mw.cfg.GetCurrentMenu()

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			r, c := row, col

			var padColor config.PadColorConfig
			if menu != nil {
				padColor = menu.Colors[r][c]
			}

			rect := canvas.NewRectangle(color.RGBA{
				R: uint8(padColor.R * 2),
				G: uint8(padColor.G * 2),
				B: uint8(padColor.B * 2),
				A: 255,
			})
			rect.SetMinSize(fyne.NewSize(40, 40))
			rect.CornerRadius = 4
			mw.gridRects[r][c] = rect

			btn := newTappableRect(rect, func() {
				mw.selectPad(r, c)
			})

			grid.Add(btn)
		}
	}

	return grid
}

func (mw *MainWindow) createColorPickerPanel() *fyne.Container {
	// Header
	header := widget.NewLabel("Pad Colors")
	header.TextStyle = fyne.TextStyle{Bold: true}

	// Create sliders for Button Color (0-127 RGB)
	mw.buttonRSlider = widget.NewSlider(0, 127)
	mw.buttonGSlider = widget.NewSlider(0, 127)
	mw.buttonBSlider = widget.NewSlider(0, 127)
	mw.buttonPreview = canvas.NewRectangle(color.RGBA{A: 255})
	mw.buttonPreview.SetMinSize(fyne.NewSize(30, 15))
	mw.buttonPreview.CornerRadius = 3

	// Create sliders for Classic Color (0-3 for R/G only, no blue)
	mw.classicRSlider = widget.NewSlider(0, 3)
	mw.classicRSlider.Step = 1
	mw.classicGSlider = widget.NewSlider(0, 3)
	mw.classicGSlider.Step = 1
	mw.classicBSlider = nil // No blue for classic
	mw.classicPreview = canvas.NewRectangle(color.RGBA{A: 255})
	mw.classicPreview.SetMinSize(fyne.NewSize(30, 15))
	mw.classicPreview.CornerRadius = 3

	// Create sliders for Pressed Color (0-127 RGB)
	mw.pressedRSlider = widget.NewSlider(0, 127)
	mw.pressedGSlider = widget.NewSlider(0, 127)
	mw.pressedBSlider = widget.NewSlider(0, 127)
	mw.pressedPreview = canvas.NewRectangle(color.RGBA{A: 255})
	mw.pressedPreview.SetMinSize(fyne.NewSize(30, 15))
	mw.pressedPreview.CornerRadius = 3

	// Create sliders for Classic Pressed Color (0-3 for R/G only)
	mw.classicPressedRSlider = widget.NewSlider(0, 3)
	mw.classicPressedRSlider.Step = 1
	mw.classicPressedGSlider = widget.NewSlider(0, 3)
	mw.classicPressedGSlider.Step = 1
	mw.classicPressedBSlider = nil // No blue for classic
	mw.classicPressedPreview = canvas.NewRectangle(color.RGBA{A: 255})
	mw.classicPressedPreview.SetMinSize(fyne.NewSize(30, 15))
	mw.classicPressedPreview.CornerRadius = 3

	// Link checkboxes - remove text to save space, just use icon/checkbox
	mw.linkButtonClassic = widget.NewCheck("", func(checked bool) {
		menu := mw.cfg.GetCurrentMenu()
		if menu != nil {
			menu.Colors[mw.selectedRow][mw.selectedCol].LinkButtonClassic = checked
			if checked {
				mw.syncClassicFromButton()
			}
			mw.setDirty(true)
		}
	})
	mw.linkButtonClassic.Checked = true

	mw.linkPressedClassic = widget.NewCheck("", func(checked bool) {
		menu := mw.cfg.GetCurrentMenu()
		if menu != nil {
			menu.Colors[mw.selectedRow][mw.selectedCol].LinkPressedClassic = checked
			if checked {
				mw.syncClassicPressedFromPressed()
			}
			mw.setDirty(true)
		}
	})
	mw.linkPressedClassic.Checked = true

	// Wire up button color slider callbacks
	buttonColorChanged := func(_ float64) { mw.onButtonColorChanged() }
	mw.buttonRSlider.OnChanged = buttonColorChanged
	mw.buttonGSlider.OnChanged = buttonColorChanged
	mw.buttonBSlider.OnChanged = buttonColorChanged

	// Wire up classic color slider callbacks
	classicColorChanged := func(_ float64) { mw.onClassicColorChanged() }
	mw.classicRSlider.OnChanged = classicColorChanged
	mw.classicGSlider.OnChanged = classicColorChanged

	// Wire up pressed color slider callbacks
	pressedColorChanged := func(_ float64) { mw.onPressedColorChanged() }
	mw.pressedRSlider.OnChanged = pressedColorChanged
	mw.pressedGSlider.OnChanged = pressedColorChanged
	mw.pressedBSlider.OnChanged = pressedColorChanged

	// Wire up classic pressed color slider callbacks
	classicPressedColorChanged := func(_ float64) { mw.onClassicPressedColorChanged() }
	mw.classicPressedRSlider.OnChanged = classicPressedColorChanged
	mw.classicPressedGSlider.OnChanged = classicPressedColorChanged

	// Helper for compact slider row with colored background
	sliderRow := func(label string, slider *widget.Slider) *fyne.Container {
		txt := canvas.NewText(label, theme.ForegroundColor())
		txt.TextSize = 10

		// Add colored background based on channel
		var bgColor color.Color
		switch label {
		case "R":
			bgColor = color.RGBA{255, 0, 0, 40} // Semi-transparent red
		case "G":
			bgColor = color.RGBA{0, 255, 0, 40} // Semi-transparent green
		case "B":
			bgColor = color.RGBA{0, 0, 255, 40} // Semi-transparent blue
		default:
			bgColor = color.Transparent
		}

		bg := canvas.NewRectangle(bgColor)
		bg.CornerRadius = 3

		sliderWithBg := container.NewStack(bg, slider)
		return container.NewBorder(nil, nil, txt, nil, sliderWithBg)
	}

	// Helper to create a rotated text image (CCW, bottom-to-top) using freetype
	rotatedLabel := func(text string) *canvas.Image {
		// Use freetype for anti-aliased rendering
		// Get font from Fyne's theme
		fontResource := theme.DefaultTextFont()
		fontBytes := fontResource.Content()

		f, err := freetype.ParseFont(fontBytes)
		if err != nil {
			log.Printf("Failed to parse font: %v", err)
			// Fallback to empty image
			return canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 1, 1)))
		}

		fontSize := float64(12)
		dpi := float64(72)

		// Create freetype context
		c := freetype.NewContext()
		c.SetFont(f)
		c.SetFontSize(fontSize)
		c.SetDPI(dpi)

		// Calculate bounds
		opts := truetype.Options{Size: fontSize, DPI: dpi}
		face := truetype.NewFace(f, &opts)
		defer face.Close()

		// Measure text width
		textWidth := 0
		for _, r := range text {
			adv, ok := face.GlyphAdvance(r)
			if ok {
				textWidth += adv.Round()
			}
		}

		metrics := face.Metrics()
		textHeight := (metrics.Ascent + metrics.Descent).Ceil()
		ascent := metrics.Ascent.Ceil()

		// Add padding
		padding := 2
		imgWidth := textWidth + padding*2
		imgHeight := textHeight + padding*2

		// Create source image
		srcImg := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

		// Get text color from theme
		textColor := theme.ForegroundColor()

		// Setup freetype context for drawing
		c.SetClip(srcImg.Bounds())
		c.SetDst(srcImg)
		c.SetSrc(image.NewUniform(textColor))

		// Draw text
		pt := freetype.Pt(padding, padding+ascent)
		_, err = c.DrawString(text, pt)
		if err != nil {
			log.Printf("Failed to draw string: %v", err)
		}

		// Create rotated image (90 deg CCW: w,h -> h,w)
		rotatedImg := image.NewRGBA(image.Rect(0, 0, imgHeight, imgWidth))

		// Rotate pixels: CCW means (x,y) -> (y, width-1-x)
		for y := 0; y < imgHeight; y++ {
			for x := 0; x < imgWidth; x++ {
				rotatedImg.Set(y, imgWidth-1-x, srcImg.At(x, y))
			}
		}

		// Create canvas image
		canvasImg := canvas.NewImageFromImage(rotatedImg)
		canvasImg.SetMinSize(fyne.NewSize(float32(imgHeight), float32(imgWidth)))
		canvasImg.FillMode = canvas.ImageFillOriginal
		return canvasImg
	}

	// --- Headers ---
	modernHeader := widget.NewLabelWithStyle("Modern", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	classicHeader := widget.NewLabelWithStyle("Classic", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	colHeaders := container.NewGridWithColumns(2, modernHeader, classicHeader)

	// spacer for header row (approx same width as rotated label height)
	headerSpacer := canvas.NewRectangle(color.Transparent)
	headerSpacer.SetMinSize(fyne.NewSize(13, 1))
	headerRow := container.NewBorder(nil, nil, headerSpacer, nil, colHeaders)

	// --- Static Section ---

	// Preview Row: [Modern] [Link] [Classic]
	staticPreviewRow := container.NewGridWithColumns(3,
		container.NewCenter(mw.buttonPreview),
		container.NewCenter(mw.linkButtonClassic),
		container.NewCenter(mw.classicPreview),
	)

	// Sliders Row: [Modern Sliders] [Classic Sliders]
	staticSlidersRow := container.NewGridWithColumns(2,
		container.NewVBox(
			sliderRow("R", mw.buttonRSlider),
			sliderRow("G", mw.buttonGSlider),
			sliderRow("B", mw.buttonBSlider),
		),
		container.NewVBox(
			sliderRow("R", mw.classicRSlider),
			sliderRow("G", mw.classicGSlider),
		),
	)

	staticContent := container.NewVBox(staticPreviewRow, staticSlidersRow)
	// Use container.NewHBox for label + content to ensure label is on left
	// Just wrapping the label in a center container to prevent stretch might act better
	staticLabel := container.NewCenter(rotatedLabel("Static"))
	staticRow := container.NewBorder(nil, nil, staticLabel, nil, staticContent)

	// --- Pressed Section ---

	// Preview Row: [Modern] [Link] [Classic]
	pressedPreviewRow := container.NewGridWithColumns(3,
		container.NewCenter(mw.pressedPreview),
		container.NewCenter(mw.linkPressedClassic),
		container.NewCenter(mw.classicPressedPreview),
	)

	// Sliders Row: [Modern Sliders] [Classic Sliders]
	pressedSlidersRow := container.NewGridWithColumns(2,
		container.NewVBox(
			sliderRow("R", mw.pressedRSlider),
			sliderRow("G", mw.pressedGSlider),
			sliderRow("B", mw.pressedBSlider),
		),
		container.NewVBox(
			sliderRow("R", mw.classicPressedRSlider),
			sliderRow("G", mw.classicPressedGSlider),
		),
	)

	pressedContent := container.NewVBox(pressedPreviewRow, pressedSlidersRow)
	pressedLabel := container.NewCenter(rotatedLabel("Pressed"))
	pressedRow := container.NewBorder(nil, nil, pressedLabel, nil, pressedContent)

	// Presets
	presetsLabel := widget.NewLabel("Presets")
	presetsLabel.TextStyle = fyne.TextStyle{Bold: true}
	presets := container.NewGridWithColumns(5,
		widget.NewButton("R", func() { mw.applyPreset(127, 0, 0) }),
		widget.NewButton("G", func() { mw.applyPreset(0, 127, 0) }),
		widget.NewButton("Y", func() { mw.applyPreset(127, 127, 0) }),
		widget.NewButton("O", func() { mw.applyPreset(127, 64, 0) }),
		widget.NewButton("⊘", func() { mw.applyPreset(0, 0, 0) }),
	)

	return container.NewVBox(
		header,
		widget.NewSeparator(),
		headerRow,
		staticRow,
		widget.NewSeparator(),
		pressedRow,
		widget.NewSeparator(),
		presets,
	)
}

func (mw *MainWindow) ensureDefaultLinking(menu *config.MenuLayout) {
	if menu == nil {
		return
	}
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			padColor := menu.Colors[r][c]
			changed := false

			// Check button color
			buttonHasColor := padColor.R > 0 || padColor.G > 0 || padColor.B > 0
			classicIsBlack := padColor.ClassicR == 0 && padColor.ClassicG == 0 && padColor.ClassicB == 0

			if buttonHasColor && classicIsBlack {
				padColor.LinkButtonClassic = true
				rLevel, gLevel := config.CalculateClassicLevel(padColor.R, padColor.G, padColor.B)
				padColor.ClassicR = config.LevelTo127(rLevel)
				padColor.ClassicG = config.LevelTo127(gLevel)
				changed = true
			}

			// Check pressed color
			pressedHasColor := padColor.PressedR > 0 || padColor.PressedG > 0 || padColor.PressedB > 0
			classicPressedIsBlack := padColor.ClassicPressedR == 0 && padColor.ClassicPressedG == 0 && padColor.ClassicPressedB == 0

			if pressedHasColor && classicPressedIsBlack {
				padColor.LinkPressedClassic = true
				rLevel, gLevel := config.CalculateClassicLevel(padColor.PressedR, padColor.PressedG, padColor.PressedB)
				padColor.ClassicPressedR = config.LevelTo127(rLevel)
				padColor.ClassicPressedG = config.LevelTo127(gLevel)
				changed = true
			}

			if changed {
				menu.Colors[r][c] = padColor
			}
		}
	}
}

func (mw *MainWindow) selectPad(row, col int) {
	mw.selectedRow = row
	mw.selectedCol = col

	// Load colors into sliders
	menu := mw.cfg.GetCurrentMenu()
	if menu == nil {
		return
	}

	// Ensure default linking for this pad (and others)
	// We could just do it for this pad, but doing it for the whole menu ensures consistency
	// However, for performance, let's just do it for this pad here, or rely on loadLayout doing it globally.
	// The user asked for "converting the colors when you load that menu or click on that specidic button".
	// So let's re-implement the single-pad logic here using the same logic as ensureDefaultLinking

	padColor := menu.Colors[row][col]
	changed := false

	buttonHasColor := padColor.R > 0 || padColor.G > 0 || padColor.B > 0
	classicIsBlack := padColor.ClassicR == 0 && padColor.ClassicG == 0 && padColor.ClassicB == 0

	if buttonHasColor && classicIsBlack {
		padColor.LinkButtonClassic = true
		rLevel, gLevel := config.CalculateClassicLevel(padColor.R, padColor.G, padColor.B)
		padColor.ClassicR = config.LevelTo127(rLevel)
		padColor.ClassicG = config.LevelTo127(gLevel)
		changed = true
	}

	pressedHasColor := padColor.PressedR > 0 || padColor.PressedG > 0 || padColor.PressedB > 0
	classicPressedIsBlack := padColor.ClassicPressedR == 0 && padColor.ClassicPressedG == 0 && padColor.ClassicPressedB == 0

	if pressedHasColor && classicPressedIsBlack {
		padColor.LinkPressedClassic = true
		rLevel, gLevel := config.CalculateClassicLevel(padColor.PressedR, padColor.PressedG, padColor.PressedB)
		padColor.ClassicPressedR = config.LevelTo127(rLevel)
		padColor.ClassicPressedG = config.LevelTo127(gLevel)
		changed = true
	}

	if changed {
		menu.Colors[row][col] = padColor
	}

	// Suppress callbacks while setting values
	mw.setSliderValues(mw.buttonRSlider, mw.buttonGSlider, mw.buttonBSlider,
		float64(padColor.R), float64(padColor.G), float64(padColor.B))

	// Use Level127To4 for classic sliders (0-127 -> 0-3)
	mw.setClassicSliderValues(mw.classicRSlider, mw.classicGSlider,
		float64(config.Level127To4(padColor.ClassicR)), float64(config.Level127To4(padColor.ClassicG)))

	mw.setSliderValues(mw.pressedRSlider, mw.pressedGSlider, mw.pressedBSlider,
		float64(padColor.PressedR), float64(padColor.PressedG), float64(padColor.PressedB))

	// Use Level127To4 for classic pressed sliders (0-127 -> 0-3)
	mw.setClassicSliderValues(mw.classicPressedRSlider, mw.classicPressedGSlider,
		float64(config.Level127To4(padColor.ClassicPressedR)), float64(config.Level127To4(padColor.ClassicPressedG)))

	// Update link checkboxes
	mw.linkButtonClassic.Checked = padColor.LinkButtonClassic
	mw.linkButtonClassic.Refresh()
	mw.linkPressedClassic.Checked = padColor.LinkPressedClassic
	mw.linkPressedClassic.Refresh()

	// Update all previews
	mw.updateAllPreviews()

	// Visual selection indicator - highlight the selected pad
	// (Simple approach: refresh grid to show selection)
	mw.refreshGridSelection()
}

func (mw *MainWindow) setSliderValues(r, g, b *widget.Slider, rv, gv, bv float64) {
	// Temporarily remove callbacks to avoid triggering updates
	rCb, gCb, bCb := r.OnChanged, g.OnChanged, b.OnChanged
	r.OnChanged, g.OnChanged, b.OnChanged = nil, nil, nil

	r.Value = rv
	g.Value = gv
	b.Value = bv
	r.Refresh()
	g.Refresh()
	b.Refresh()

	// Restore callbacks
	r.OnChanged, g.OnChanged, b.OnChanged = rCb, gCb, bCb
}

func (mw *MainWindow) setClassicSliderValues(r, g *widget.Slider, rv, gv float64) {
	// Temporarily remove callbacks to avoid triggering updates
	rCb, gCb := r.OnChanged, g.OnChanged
	r.OnChanged, g.OnChanged = nil, nil

	r.Value = rv
	g.Value = gv
	r.Refresh()
	g.Refresh()

	// Restore callbacks
	r.OnChanged, g.OnChanged = rCb, gCb
}

func (mw *MainWindow) updateAllPreviews() {
	mw.updateButtonPreview()
	mw.updateClassicPreview()
	mw.updatePressedPreview()
	mw.updateClassicPressedPreview()
}

func (mw *MainWindow) updateButtonPreview() {
	mw.buttonPreview.FillColor = color.RGBA{
		R: uint8(mw.buttonRSlider.Value * 2),
		G: uint8(mw.buttonGSlider.Value * 2),
		B: uint8(mw.buttonBSlider.Value * 2),
		A: 255,
	}
	mw.buttonPreview.Refresh()
}

func (mw *MainWindow) updateClassicPreview() {
	// Convert 0-3 levels to display RGB (0, 85, 170, 255)
	mw.classicPreview.FillColor = color.RGBA{
		R: uint8(mw.classicRSlider.Value * 85),
		G: uint8(mw.classicGSlider.Value * 85),
		B: 0, // No blue for classic
		A: 255,
	}
	mw.classicPreview.Refresh()
}

func (mw *MainWindow) updatePressedPreview() {
	mw.pressedPreview.FillColor = color.RGBA{
		R: uint8(mw.pressedRSlider.Value * 2),
		G: uint8(mw.pressedGSlider.Value * 2),
		B: uint8(mw.pressedBSlider.Value * 2),
		A: 255,
	}
	mw.pressedPreview.Refresh()
}

func (mw *MainWindow) updateClassicPressedPreview() {
	// Convert 0-3 levels to display RGB (0, 85, 170, 255)
	mw.classicPressedPreview.FillColor = color.RGBA{
		R: uint8(mw.classicPressedRSlider.Value * 85),
		G: uint8(mw.classicPressedGSlider.Value * 85),
		B: 0, // No blue for classic
		A: 255,
	}
	mw.classicPressedPreview.Refresh()
}

func (mw *MainWindow) onButtonColorChanged() {
	mw.updateButtonPreview()
	mw.saveCurrentPadColors()

	// If linked, update classic color
	if mw.linkButtonClassic.Checked {
		mw.syncClassicFromButton()
	}

	// Update grid display
	mw.updateGridRect(mw.selectedRow, mw.selectedCol)
	mw.setDirty(true)
}

func (mw *MainWindow) onClassicColorChanged() {
	// When classic is manually changed, unlink
	if mw.linkButtonClassic.Checked {
		mw.linkButtonClassic.Checked = false
		mw.linkButtonClassic.Refresh()
		menu := mw.cfg.GetCurrentMenu()
		if menu != nil {
			menu.Colors[mw.selectedRow][mw.selectedCol].LinkButtonClassic = false
		}
	}
	mw.updateClassicPreview()
	mw.saveCurrentPadColors()
	mw.setDirty(true)
}

func (mw *MainWindow) onPressedColorChanged() {
	mw.updatePressedPreview()
	mw.saveCurrentPadColors()

	// If linked, update classic pressed color
	if mw.linkPressedClassic.Checked {
		mw.syncClassicPressedFromPressed()
	}
	mw.setDirty(true)
}

func (mw *MainWindow) onClassicPressedColorChanged() {
	// When classic pressed is manually changed, unlink
	if mw.linkPressedClassic.Checked {
		mw.linkPressedClassic.Checked = false
		mw.linkPressedClassic.Refresh()
		menu := mw.cfg.GetCurrentMenu()
		if menu != nil {
			menu.Colors[mw.selectedRow][mw.selectedCol].LinkPressedClassic = false
		}
	}
	mw.updateClassicPressedPreview()
	mw.saveCurrentPadColors()
	mw.setDirty(true)
}

func (mw *MainWindow) syncClassicFromButton() {
	rLevel, gLevel := config.CalculateClassicLevel(
		uint8(mw.buttonRSlider.Value),
		uint8(mw.buttonGSlider.Value),
		uint8(mw.buttonBSlider.Value),
	)
	mw.setClassicSliderValues(mw.classicRSlider, mw.classicGSlider,
		float64(rLevel), float64(gLevel))
	mw.updateClassicPreview()
	mw.saveCurrentPadColors()
}

func (mw *MainWindow) syncClassicPressedFromPressed() {
	rLevel, gLevel := config.CalculateClassicLevel(
		uint8(mw.pressedRSlider.Value),
		uint8(mw.pressedGSlider.Value),
		uint8(mw.pressedBSlider.Value),
	)
	mw.setClassicSliderValues(mw.classicPressedRSlider, mw.classicPressedGSlider,
		float64(rLevel), float64(gLevel))
	mw.updateClassicPressedPreview()
	mw.saveCurrentPadColors()
}

func (mw *MainWindow) saveCurrentPadColors() {
	menu := mw.cfg.GetCurrentMenu()
	if menu == nil {
		return
	}

	menu.Colors[mw.selectedRow][mw.selectedCol] = config.PadColorConfig{
		R: uint8(mw.buttonRSlider.Value),
		G: uint8(mw.buttonGSlider.Value),
		B: uint8(mw.buttonBSlider.Value),

		// Convert 0-3 levels to 0-127 for storage
		ClassicR: config.LevelTo127(uint8(mw.classicRSlider.Value)),
		ClassicG: config.LevelTo127(uint8(mw.classicGSlider.Value)),
		ClassicB: 0, // No blue for classic

		PressedR: uint8(mw.pressedRSlider.Value),
		PressedG: uint8(mw.pressedGSlider.Value),
		PressedB: uint8(mw.pressedBSlider.Value),

		// Convert 0-3 levels to 0-127 for storage
		ClassicPressedR: config.LevelTo127(uint8(mw.classicPressedRSlider.Value)),
		ClassicPressedG: config.LevelTo127(uint8(mw.classicPressedGSlider.Value)),
		ClassicPressedB: 0, // No blue for classic

		LinkButtonClassic:  mw.linkButtonClassic.Checked,
		LinkPressedClassic: mw.linkPressedClassic.Checked,
	}
}

func (mw *MainWindow) applyPreset(r, g, b float64) {
	mw.setSliderValues(mw.buttonRSlider, mw.buttonGSlider, mw.buttonBSlider, r, g, b)
	mw.onButtonColorChanged()
}

func (mw *MainWindow) refreshGridSelection() {
	// Highlight selected pad by updating its visual appearance
	// For now, just refresh the grid - selection is implicit from color panel content
	mw.refreshGrid()
}

func (mw *MainWindow) updateGridRect(row, col int) {
	menu := mw.cfg.GetCurrentMenu()
	if menu == nil {
		return
	}
	c := menu.Colors[row][col]
	mw.gridRects[row][col].FillColor = color.RGBA{
		R: uint8(c.R * 2),
		G: uint8(c.G * 2),
		B: uint8(c.B * 2),
		A: 255,
	}
	mw.gridRects[row][col].Refresh()
}

func (mw *MainWindow) setDirty(dirty bool) {
	mw.dirty = dirty
	if mw.revertBtn != nil {
		if dirty {
			mw.revertBtn.Enable()
		} else {
			mw.revertBtn.Disable()
		}
	}
}

func (mw *MainWindow) revertLayout() {
	// Reload from disk
	if savedCfg, err := config.Load(); err == nil {
		mw.cfg.Menus = savedCfg.Menus
	}
	mw.setDirty(false)
	mw.refreshGrid()
}

func (mw *MainWindow) clearGrid() {
	menu := mw.cfg.GetCurrentMenu()
	if menu == nil {
		return
	}
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			menu.Colors[row][col] = config.PadColorConfig{R: 0, G: 0, B: 0}
			mw.updateGridRect(row, col)
		}
	}
	mw.setDirty(true)
}

func (mw *MainWindow) saveLayout() {
	if err := mw.cfg.Save(); err != nil {
		log.Printf("Failed to save layout: %v", err)
	} else {
		log.Printf("Layout saved")
		mw.setDirty(false)
		// Apply to devices after save
		mw.sendGridToDevices()
	}
}

func (mw *MainWindow) saveAsNewLayout() {
	currentMenu := mw.cfg.GetCurrentMenu()
	if currentMenu == nil {
		return
	}

	entry := widget.NewEntry()
	entry.SetPlaceHolder("Layout Name")
	entry.SetText(currentMenu.Name + " Copy")

	dialog.ShowCustomConfirm("Save As New Layout", "Save", "Cancel",
		container.NewVBox(widget.NewLabel("Enter a name for the new layout:"), entry),
		func(confirm bool) {
			if confirm && entry.Text != "" {
				// Create new layout with copied colors
				newMenu := config.NewMenuLayout()
				newMenu.Name = entry.Text
				newMenu.Colors = currentMenu.Colors // Copy colors

				mw.cfg.Menus = append(mw.cfg.Menus, newMenu)
				mw.cfg.CurrentMenuID = newMenu.ID
				mw.layoutDropdown.Options = mw.getLayoutNames()
				mw.layoutDropdown.SetSelected(newMenu.Name)
				mw.cfg.Save()
				mw.sendGridToDevices()
			}
		}, mw.window)
}

func (mw *MainWindow) sendGridToDevices() {
	for _, device := range mw.cfg.Devices {
		if device.OutPort == "" {
			continue
		}

		// Find the menu assigned to this device
		var menu *config.MenuLayout
		if device.MainMenu == "" {
			// No menu assigned, skip this device
			continue
		}
		for i := range mw.cfg.Menus {
			if mw.cfg.Menus[i].Name == device.MainMenu {
				menu = &mw.cfg.Menus[i]
				break
			}
		}
		if menu == nil {
			log.Printf("Menu '%s' not found for device %s", device.MainMenu, device.Name)
			continue
		}

		// Ensure legacy/uninitialized colors are linked and converted before sending
		mw.ensureDefaultLinking(menu)

		deviceType := midi.DeviceType(device.Type)

		for row := 0; row < 9; row++ {
			for col := 0; col < 9; col++ {
				c := menu.Colors[row][col]

				// Use classic colors for classic devices, button colors for colorful devices
				var padColor midi.PadColor
				if deviceType == midi.DeviceTypeClassic {
					padColor = midi.PadColor{R: c.ClassicR, G: c.ClassicG, B: c.ClassicB}
				} else {
					padColor = midi.PadColor{R: c.R, G: c.G, B: c.B}
				}

				if err := mw.midiManager.SetPadColor(device.OutPort, deviceType, row, col, padColor); err != nil {
					log.Printf("Failed to set pad color: %v", err)
				}
			}
		}
		log.Printf("Sent layout '%s' to %s", menu.Name, device.Name)
	}
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

// ============ TAPPABLE RECTANGLE WIDGET ============

type tappableRect struct {
	widget.BaseWidget
	rect  *canvas.Rectangle
	onTap func()
}

func newTappableRect(rect *canvas.Rectangle, onTap func()) *tappableRect {
	t := &tappableRect{rect: rect, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableRect) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.rect)
}

func (t *tappableRect) Tapped(_ *fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tappableRect) TappedSecondary(_ *fyne.PointEvent) {}
