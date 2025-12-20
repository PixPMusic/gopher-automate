package window

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/PixPMusic/gopher-automate/internal/config"
)

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

	typeSelect := widget.NewSelect([]string{"Classic", "Colorful", "Generic"}, nil)
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
	case config.DeviceTypeGeneric:
		typeSelect.SetSelected("Generic")
	default:
		typeSelect.SetSelected("Classic")
	}
	typeSelect.OnChanged = func(s string) {
		switch s {
		case "Classic":
			device.Type = config.DeviceTypeClassic
			menuSelect.Enable()
		case "Colorful":
			device.Type = config.DeviceTypeColorful
			menuSelect.Enable()
		case "Generic":
			device.Type = config.DeviceTypeGeneric
			device.MainMenu = "" // Clear menu assignment
			menuSelect.SetSelected("(None)")
			menuSelect.Disable()
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

	// Disable menu dropdown for Generic devices
	if device.Type == config.DeviceTypeGeneric {
		menuSelect.Disable()
	} else {
		menuSelect.Enable()
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
