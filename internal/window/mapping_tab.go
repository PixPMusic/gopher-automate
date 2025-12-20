package window

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/PixPMusic/gopher-automate/internal/config"
)

// ============ MESSAGE MAPPING TAB ============

func (mw *MainWindow) createMessageMappingTab() fyne.CanvasObject {
	header := widget.NewLabel("Message Mapping")
	header.TextStyle = fyne.TextStyle{Bold: true}

	subtitle := widget.NewLabel("Map MIDI messages from Generic devices to actions (inter-app communication)")

	// Column headers
	headerName := widget.NewLabel("Name")
	headerName.TextStyle = fyne.TextStyle{Bold: true}
	headerType := widget.NewLabel("Type")
	headerType.TextStyle = fyne.TextStyle{Bold: true}
	headerChannel := widget.NewLabel("Channel")
	headerChannel.TextStyle = fyne.TextStyle{Bold: true}
	headerNumber := widget.NewLabel("Number")
	headerNumber.TextStyle = fyne.TextStyle{Bold: true}
	headerAction := widget.NewLabel("Action")
	headerAction.TextStyle = fyne.TextStyle{Bold: true}
	headerDelete := widget.NewLabel("")

	columnHeaders := container.NewGridWithColumns(6,
		headerName, headerType, headerChannel, headerNumber, headerAction, headerDelete,
	)

	// Create the mapping list
	mw.mappingList = widget.NewList(
		func() int {
			return len(mw.cfg.MessageMappings)
		},
		func() fyne.CanvasObject {
			return mw.createMappingListItem()
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			mw.updateMappingListItem(id, obj)
		},
	)

	// Toolbar
	addBtn := widget.NewButtonWithIcon("Add Mapping", theme.ContentAddIcon(), func() {
		mw.addMessageMapping()
	})
	listToolbar := container.NewHBox(addBtn)

	// Save button
	saveBtn := widget.NewButtonWithIcon("Save Mappings", theme.DocumentSaveIcon(), func() {
		mw.saveMessageMappings()
	})
	saveBtn.Importance = widget.HighImportance

	return container.NewBorder(
		container.NewVBox(header, subtitle, widget.NewSeparator(), listToolbar, columnHeaders),
		container.NewVBox(widget.NewSeparator(), container.NewHBox(saveBtn)),
		nil, nil,
		mw.mappingList,
	)
}

func (mw *MainWindow) createMappingListItem() fyne.CanvasObject {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Mapping name")

	typeSelect := widget.NewSelect([]string{"Note", "CC", "Program Change"}, nil)
	typeSelect.PlaceHolder = "Type"

	channelSelect := widget.NewSelect([]string{"Any", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16"}, nil)
	channelSelect.PlaceHolder = "Ch"

	numberEntry := widget.NewEntry()
	numberEntry.SetPlaceHolder("0-127")

	actionSelect := widget.NewSelect([]string{"(None)"}, nil)
	actionSelect.PlaceHolder = "Action"

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)

	return container.NewGridWithColumns(6, nameEntry, typeSelect, channelSelect, numberEntry, actionSelect, deleteBtn)
}

func (mw *MainWindow) updateMappingListItem(id widget.ListItemID, obj fyne.CanvasObject) {
	if id >= len(mw.cfg.MessageMappings) {
		return
	}

	mapping := &mw.cfg.MessageMappings[id]
	row := obj.(*fyne.Container)

	nameEntry := row.Objects[0].(*widget.Entry)
	typeSelect := row.Objects[1].(*widget.Select)
	channelSelect := row.Objects[2].(*widget.Select)
	numberEntry := row.Objects[3].(*widget.Entry)
	actionSelect := row.Objects[4].(*widget.Select)
	deleteBtn := row.Objects[5].(*widget.Button)

	// Set up delete button
	mappingID := mapping.ID
	deleteBtn.OnTapped = func() {
		mw.deleteMappingByID(mappingID)
	}

	// Set up name entry
	nameEntry.SetText(mapping.Name)
	nameEntry.OnChanged = func(s string) {
		mapping.Name = s
	}

	// Set up message type
	switch mapping.MessageType {
	case "note":
		typeSelect.SetSelected("Note")
	case "cc":
		typeSelect.SetSelected("CC")
	case "program_change":
		typeSelect.SetSelected("Program Change")
	}
	typeSelect.OnChanged = func(s string) {
		switch s {
		case "Note":
			mapping.MessageType = "note"
		case "CC":
			mapping.MessageType = "cc"
		case "Program Change":
			mapping.MessageType = "program_change"
		}
	}

	// Set up channel
	if mapping.Channel == -1 {
		channelSelect.SetSelected("Any")
	} else {
		channelSelect.SetSelected(fmt.Sprintf("%d", mapping.Channel+1))
	}
	channelSelect.OnChanged = func(s string) {
		if s == "Any" {
			mapping.Channel = -1
		} else {
			var ch int
			fmt.Sscanf(s, "%d", &ch)
			mapping.Channel = ch - 1
		}
	}

	// Set up number
	numberEntry.SetText(fmt.Sprintf("%d", mapping.Number))
	numberEntry.OnChanged = func(s string) {
		var num int
		if _, err := fmt.Sscanf(s, "%d", &num); err == nil && num >= 0 && num <= 127 {
			mapping.Number = num
		}
	}

	// Set up action dropdown
	mw.refreshMappingActionOptions(actionSelect)
	if mapping.ActionID == "" {
		actionSelect.SetSelected("(None)")
	} else {
		action := mw.cfg.GetAction(mapping.ActionID)
		if action != nil {
			actionSelect.SetSelected(action.Name)
		} else {
			actionSelect.SetSelected("(None)")
		}
	}
	actionSelect.OnChanged = func(s string) {
		if s == "(None)" {
			mapping.ActionID = ""
		} else {
			// Find action by name
			for _, a := range mw.cfg.Actions {
				if a.Name == s {
					mapping.ActionID = a.ID
					break
				}
			}
		}
	}
}

func (mw *MainWindow) refreshMappingActionOptions(actionSelect *widget.Select) {
	options := []string{"(None)"}
	for _, a := range mw.cfg.Actions {
		options = append(options, a.Name)
	}
	actionSelect.Options = options
}

func (mw *MainWindow) addMessageMapping() {
	mapping := config.NewMessageMapping()
	mw.cfg.MessageMappings = append(mw.cfg.MessageMappings, mapping)
	mw.mappingList.Refresh()
}

func (mw *MainWindow) deleteMappingByID(id string) {
	// Find the mapping by ID
	for i, m := range mw.cfg.MessageMappings {
		if m.ID == id {
			dialog.ShowConfirm("Delete Mapping", "Are you sure you want to delete '"+m.Name+"'?",
				func(confirm bool) {
					if confirm {
						mw.cfg.MessageMappings = append(
							mw.cfg.MessageMappings[:i],
							mw.cfg.MessageMappings[i+1:]...,
						)
						mw.mappingList.Refresh()
					}
				}, mw.window)
			return
		}
	}
}

func (mw *MainWindow) saveMessageMappings() {
	if err := mw.cfg.Save(); err != nil {
		log.Printf("Failed to save message mappings: %v", err)
		dialog.ShowError(err, mw.window)
	} else {
		dialog.ShowInformation("Saved", "Message mappings saved successfully.", mw.window)
	}
}
