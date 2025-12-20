package window

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/PixPMusic/gopher-automate/internal/actions"
)

// ============ ACTIONS TAB ============

func (mw *MainWindow) createActionsTab() fyne.CanvasObject {
	header := widget.NewLabel("Actions")
	header.TextStyle = fyne.TextStyle{Bold: true}

	subtitle := widget.NewLabel("Create and manage executable actions")

	// Create the action list
	mw.actionList = widget.NewList(
		func() int {
			return len(mw.actionStore.GetFlatList())
		},
		func() fyne.CanvasObject {
			return mw.createActionListItem()
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			mw.updateActionListItem(id, obj)
		},
	)
	mw.actionList.OnSelected = func(id widget.ListItemID) {
		mw.selectActionItem(id)
	}

	// Action list toolbar
	addGroupBtn := widget.NewButtonWithIcon("Add Group", theme.FolderNewIcon(), func() {
		mw.addActionGroup()
	})
	addActionBtn := widget.NewButtonWithIcon("Add Action", theme.ContentAddIcon(), func() {
		mw.addAction()
	})
	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		mw.deleteSelectedActionItem()
	})
	moveUpBtn := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {
		mw.moveSelectedActionUp()
	})
	moveDownBtn := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
		mw.moveSelectedActionDown()
	})
	listToolbar := container.NewHBox(addGroupBtn, addActionBtn, deleteBtn, layout.NewSpacer(), moveUpBtn, moveDownBtn)

	listPanel := container.NewBorder(
		listToolbar,
		nil, nil, nil,
		mw.actionList,
	)

	// Create the action editor panel
	mw.actionEditor = mw.createActionEditorPanel()

	// Horizontal split: list on left, editor on right
	split := container.NewHSplit(listPanel, container.NewVScroll(mw.actionEditor))
	split.Offset = 0.35

	// Save button
	saveBtn := widget.NewButtonWithIcon("Save Actions", theme.DocumentSaveIcon(), func() {
		mw.saveActions()
	})
	saveBtn.Importance = widget.HighImportance

	return container.NewBorder(
		container.NewVBox(header, subtitle, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), container.NewHBox(saveBtn)),
		nil, nil,
		split,
	)
}

func (mw *MainWindow) createActionListItem() fyne.CanvasObject {
	icon := widget.NewIcon(theme.DocumentIcon())
	name := widget.NewLabel("Action Name")
	typeLabel := widget.NewLabel("")
	typeLabel.TextStyle = fyne.TextStyle{Italic: true}

	return container.NewHBox(icon, name, typeLabel)
}

func (mw *MainWindow) updateActionListItem(id widget.ListItemID, obj fyne.CanvasObject) {
	items := mw.actionStore.GetFlatList()
	if id >= len(items) {
		return
	}

	item := items[id]
	row := obj.(*fyne.Container)
	icon := row.Objects[0].(*widget.Icon)
	name := row.Objects[1].(*widget.Label)
	typeLabel := row.Objects[2].(*widget.Label)

	// Add indentation based on depth
	indent := strings.Repeat("  ", item.Depth)

	if item.IsGroup {
		icon.SetResource(theme.FolderIcon())
		name.SetText(indent + item.Group.Name)
		typeLabel.SetText("")
	} else {
		icon.SetResource(theme.DocumentIcon())
		name.SetText(indent + item.Action.Name)
		switch item.Action.Type {
		case actions.ActionTypeAppleScript:
			typeLabel.SetText("(AppleScript)")
		case actions.ActionTypeShellCommand:
			typeLabel.SetText("(" + mw.executor.GetShellName() + ")")
		case actions.ActionTypeSleep:
			typeLabel.SetText("(Sleep)")
		case actions.ActionTypeMidi:
			typeLabel.SetText("(MIDI)")
		}
	}
}

func (mw *MainWindow) selectActionItem(id widget.ListItemID) {
	items := mw.actionStore.GetFlatList()
	if id >= len(items) {
		return
	}

	item := items[id]
	if item.IsGroup {
		mw.selectedGroup = item.Group
		mw.selectedAction = nil
		mw.updateActionEditor()
	} else {
		mw.selectedAction = item.Action
		mw.selectedGroup = nil
		mw.updateActionEditor()
	}
}

func (mw *MainWindow) createActionEditorPanel() *fyne.Container {
	header := widget.NewLabel("Editor")
	header.TextStyle = fyne.TextStyle{Bold: true}

	// Name entry
	nameLabel := widget.NewLabel("Name:")
	mw.actionNameEntry = widget.NewEntry()
	mw.actionNameEntry.SetPlaceHolder("Action name")
	mw.actionNameEntry.OnChanged = func(s string) {
		if mw.selectedAction != nil {
			mw.selectedAction.Name = s
			mw.actionStore.UpdateAction(mw.selectedAction)
			mw.actionList.Refresh()
		} else if mw.selectedGroup != nil {
			mw.selectedGroup.Name = s
			mw.actionStore.UpdateGroup(mw.selectedGroup)
			mw.actionList.Refresh()
		}
	}

	// Wait for Completion Checkbox
	mw.waitForCompletionCheck = widget.NewCheck("Wait for completion", func(checked bool) {
		if mw.selectedAction != nil {
			mw.selectedAction.WaitForCompletion = checked
			mw.actionStore.UpdateAction(mw.selectedAction)
		}
	})

	// Type selector (only for actions)
	typeLabel := widget.NewLabel("Type:")
	typeOptions := []string{"Shell Command", "Sleep", "Send MIDI Message"}
	if mw.executor.CanExecuteAppleScript() {
		typeOptions = append([]string{"AppleScript"}, typeOptions...)
	}
	mw.actionTypeSelect = widget.NewSelect(typeOptions, func(s string) {
		if mw.selectedAction != nil {
			switch s {
			case "AppleScript":
				mw.selectedAction.Type = actions.ActionTypeAppleScript
			case "Shell Command":
				mw.selectedAction.Type = actions.ActionTypeShellCommand
			case "Sleep":
				mw.selectedAction.Type = actions.ActionTypeSleep
			case "Send MIDI Message":
				mw.selectedAction.Type = actions.ActionTypeMidi
			}
			mw.actionStore.UpdateAction(mw.selectedAction)
			// Re-update editor to show correct fields for new type
			mw.updateActionEditor()
		}
	})

	// --- Code Editor Fields (Scripting) ---
	mw.actionCodeEntry = widget.NewMultiLineEntry()
	mw.actionCodeEntry.SetPlaceHolder("Enter your script or command here...")
	mw.actionCodeEntry.SetMinRowsVisible(8)
	mw.actionCodeEntry.OnChanged = func(s string) {
		if mw.selectedAction != nil {
			mw.selectedAction.Code = s
			mw.actionStore.UpdateAction(mw.selectedAction)
			mw.updateCodePreview()
		}
	}

	// Syntax-highlighted preview label
	initialPreview := widget.NewRichText(&widget.TextSegment{Text: "(no code)"})
	mw.codePreviewScroll = container.NewVScroll(initialPreview)
	mw.codePreviewScroll.SetMinSize(fyne.NewSize(0, 100))

	// --- Sleep Editor Fields ---
	mw.sleepDurationEntry = widget.NewEntry()
	mw.sleepDurationEntry.SetPlaceHolder("Duration in seconds (e.g. 2.5)")
	mw.sleepDurationEntry.OnChanged = func(s string) {
		if mw.selectedAction != nil && mw.selectedAction.Type == actions.ActionTypeSleep {
			mw.selectedAction.Code = s
			mw.actionStore.UpdateAction(mw.selectedAction)
		}
	}

	// --- MIDI Editor Fields ---
	// Devices: We need a way to refresh this list dynamically
	mw.midiDeviceSelect = widget.NewSelect([]string{}, func(s string) {
		mw.updateMidiJSON()
	})
	mw.midiDeviceSelect.PlaceHolder = "Select Target Device"

	mw.midiMsgTypeSelect = widget.NewRadioGroup([]string{"Note On", "Note Off", "CC", "PC", "SysEx"}, func(s string) {
		// Update visibility of params based on type
		mw.updateMidiEditorVisibility(s)
		mw.updateMidiJSON()
	})
	mw.midiMsgTypeSelect.Horizontal = true

	// Channels 1-16
	channels := make([]string, 16)
	for i := 0; i < 16; i++ {
		channels[i] = fmt.Sprintf("%d", i+1)
	}
	mw.midiChannelSelect = widget.NewSelect(channels, func(s string) {
		mw.updateMidiJSON()
	})

	mw.midiNoteEntry = widget.NewEntry()
	mw.midiNoteEntry.SetPlaceHolder("0-127")
	mw.midiNoteEntry.OnChanged = func(s string) { mw.updateMidiJSON() }

	mw.midiVelocityEntry = widget.NewEntry()
	mw.midiVelocityEntry.SetPlaceHolder("0-127")
	mw.midiVelocityEntry.OnChanged = func(s string) { mw.updateMidiJSON() }

	mw.midiProgramEntry = widget.NewEntry()
	mw.midiProgramEntry.SetPlaceHolder("0-127")
	mw.midiProgramEntry.OnChanged = func(s string) { mw.updateMidiJSON() }

	mw.midiSysexEntry = widget.NewMultiLineEntry()
	mw.midiSysexEntry.SetPlaceHolder("Hex bytes (e.g. F0 01 02 F7)")
	mw.midiSysexEntry.OnChanged = func(s string) { mw.updateMidiJSON() }

	// Main container that will hold the swappable content
	mw.actionEditorContent = container.NewVBox()

	// Feedback label
	mw.actionFeedback = widget.NewLabel("")
	mw.actionFeedback.Wrapping = fyne.TextWrapWord

	// Test button
	testBtn := widget.NewButtonWithIcon("Test", theme.MediaPlayIcon(), func() {
		mw.testAction()
	})

	// Validate button
	validateBtn := widget.NewButtonWithIcon("Validate", theme.ConfirmIcon(), func() {
		mw.validateAction()
	})

	actionButtons := container.NewHBox(validateBtn, testBtn)

	return container.NewVBox(
		header,
		widget.NewSeparator(),
		container.NewBorder(nil, nil, nameLabel, nil, mw.actionNameEntry),
		container.NewBorder(nil, nil, typeLabel, nil, mw.actionTypeSelect),
		mw.waitForCompletionCheck,
		widget.NewSeparator(),
		mw.actionEditorContent, // Dynamic content
		widget.NewSeparator(),
		actionButtons,
		mw.actionFeedback,
	)
}

func (mw *MainWindow) updateCodePreview() {
	if mw.codePreviewScroll == nil {
		return
	}

	if mw.selectedAction == nil {
		mw.codePreviewScroll.Content = widget.NewRichText(&widget.TextSegment{
			Text:  "(no action selected)",
			Style: widget.RichTextStyle{Inline: true, TextStyle: fyne.TextStyle{Italic: true}},
		})
	} else {
		preview := mw.syntaxHighlighter.HighlightCode(mw.selectedAction.Code, mw.selectedAction.Type)
		preview.Wrapping = fyne.TextWrapWord
		mw.codePreviewScroll.Content = preview
	}
	mw.codePreviewScroll.Refresh()
}

func (mw *MainWindow) updateActionEditor() {
	mw.actionEditorContent.Objects = nil // Clear current content

	if mw.selectedAction != nil {
		mw.actionNameEntry.OnChanged = nil // Disable callback
		mw.actionNameEntry.SetText(mw.selectedAction.Name)
		mw.actionNameEntry.OnChanged = func(s string) {
			if mw.selectedAction != nil {
				mw.selectedAction.Name = s
				mw.actionStore.UpdateAction(mw.selectedAction)
				mw.actionList.Refresh()
			}
		}
		mw.actionNameEntry.Enable()
		mw.actionTypeSelect.Enable()
		mw.waitForCompletionCheck.Show()
		mw.waitForCompletionCheck.SetChecked(mw.selectedAction.WaitForCompletion)

		mw.actionTypeSelect.OnChanged = nil // Disable callback
		switch mw.selectedAction.Type {
		case actions.ActionTypeAppleScript:
			mw.actionTypeSelect.SetSelected("AppleScript")
			mw.showScriptEditor()
		case actions.ActionTypeShellCommand:
			mw.actionTypeSelect.SetSelected("Shell Command")
			mw.showScriptEditor()
		case actions.ActionTypeSleep:
			mw.actionTypeSelect.SetSelected("Sleep")
			mw.showSleepEditor()
		case actions.ActionTypeMidi:
			mw.actionTypeSelect.SetSelected("Send MIDI Message")
			mw.showMidiEditor()
		}
		mw.actionTypeSelect.OnChanged = func(s string) {
			if mw.selectedAction != nil {
				switch s {
				case "AppleScript":
					mw.selectedAction.Type = actions.ActionTypeAppleScript
				case "Shell Command":
					mw.selectedAction.Type = actions.ActionTypeShellCommand
				case "Sleep":
					mw.selectedAction.Type = actions.ActionTypeSleep
				case "Send MIDI Message":
					mw.selectedAction.Type = actions.ActionTypeMidi
				}
				mw.actionStore.UpdateAction(mw.selectedAction)
				mw.updateActionEditor()
			}
		}

		mw.actionFeedback.SetText("")
	} else if mw.selectedGroup != nil {
		// Group Editor
		mw.actionNameEntry.OnChanged = nil
		mw.actionNameEntry.SetText(mw.selectedGroup.Name)
		mw.actionNameEntry.OnChanged = func(s string) {
			if mw.selectedGroup != nil {
				mw.selectedGroup.Name = s
				mw.actionStore.UpdateGroup(mw.selectedGroup)
				mw.actionList.Refresh()
			}
		}
		mw.actionNameEntry.Enable()
		mw.actionTypeSelect.Disable()
		mw.waitForCompletionCheck.Hide()

		mw.actionFeedback.SetText("Groups contain actions. Select an action to edit.")
	} else {
		// Nothing selected
		mw.actionNameEntry.OnChanged = nil
		mw.actionNameEntry.SetText("")
		mw.actionNameEntry.Disable()
		mw.actionTypeSelect.Disable()
		mw.waitForCompletionCheck.Hide()

		mw.actionFeedback.SetText("Select an action or group")
	}

	mw.actionEditorContent.Refresh()
}

func (mw *MainWindow) showScriptEditor() {
	mw.actionCodeEntry.OnChanged = nil
	mw.actionCodeEntry.SetText(mw.selectedAction.Code)
	mw.actionCodeEntry.OnChanged = func(s string) {
		if mw.selectedAction != nil {
			mw.selectedAction.Code = s
			mw.actionStore.UpdateAction(mw.selectedAction)
			mw.updateCodePreview()
		}
	}
	mw.updateCodePreview()

	mw.actionEditorContent.Add(widget.NewLabel("Code:"))
	mw.actionEditorContent.Add(mw.actionCodeEntry)
	mw.actionEditorContent.Add(widget.NewLabel("Preview:"))
	mw.actionEditorContent.Add(mw.codePreviewScroll)
}

func (mw *MainWindow) showSleepEditor() {
	mw.sleepDurationEntry.OnChanged = nil
	mw.sleepDurationEntry.SetText(mw.selectedAction.Code)
	mw.sleepDurationEntry.OnChanged = func(s string) {
		if mw.selectedAction != nil && mw.selectedAction.Type == actions.ActionTypeSleep {
			mw.selectedAction.Code = s
			mw.actionStore.UpdateAction(mw.selectedAction)
		}
	}
	mw.actionEditorContent.Add(container.NewBorder(nil, nil, widget.NewLabel("Delay (seconds):"), nil, mw.sleepDurationEntry))
}

func (mw *MainWindow) showMidiEditor() {
	// Need to check current Code value and populate fields
	var data actions.MidiActionData
	if mw.selectedAction.Code != "" {
		_ = json.Unmarshal([]byte(mw.selectedAction.Code), &data)
	}

	// Refresh device list
	devices := mw.midiManager.ListOutPorts()
	// Add configured named devices too if not present
	for _, dev := range mw.cfg.Devices {
		found := false
		for _, d := range devices {
			if d == dev.Name {
				found = true
				break
			}
		}
		if !found {
			devices = append(devices, dev.Name)
		}
	}
	mw.midiDeviceSelect.Options = devices
	mw.midiDeviceSelect.OnChanged = nil
	mw.midiDeviceSelect.SetSelected(data.DeviceName)
	mw.midiDeviceSelect.OnChanged = func(s string) { mw.updateMidiJSON() }

	// Set Msg Type
	displayType := "Note On"
	switch data.MsgType {
	case "note_on":
		displayType = "Note On"
	case "note_off":
		displayType = "Note Off"
	case "cc":
		displayType = "CC"
	case "pc":
		displayType = "PC"
	case "sysex":
		displayType = "SysEx"
	}

	mw.midiMsgTypeSelect.OnChanged = nil
	mw.midiMsgTypeSelect.SetSelected(displayType) // This triggers updateMidiEditorVisibility via OnChanged
	mw.midiMsgTypeSelect.OnChanged = func(s string) {
		// Update visibility of params based on type
		mw.updateMidiEditorVisibility(s)
		mw.updateMidiJSON()
	}

	// Set Params
	mw.midiChannelSelect.OnChanged = nil
	if data.Channel > 0 {
		mw.midiChannelSelect.SetSelected(fmt.Sprintf("%d", data.Channel))
	} else {
		mw.midiChannelSelect.SetSelected("1")
	}
	mw.midiChannelSelect.OnChanged = func(s string) { mw.updateMidiJSON() }

	mw.midiNoteEntry.OnChanged = nil
	mw.midiNoteEntry.SetText(fmt.Sprintf("%d", data.Note))
	mw.midiNoteEntry.OnChanged = func(s string) { mw.updateMidiJSON() }

	mw.midiVelocityEntry.OnChanged = nil
	mw.midiVelocityEntry.SetText(fmt.Sprintf("%d", data.Velocity))
	mw.midiVelocityEntry.OnChanged = func(s string) { mw.updateMidiJSON() }

	mw.midiProgramEntry.OnChanged = nil
	mw.midiProgramEntry.SetText(fmt.Sprintf("%d", data.Program))
	mw.midiProgramEntry.OnChanged = func(s string) { mw.updateMidiJSON() }

	mw.midiSysexEntry.OnChanged = nil
	mw.midiSysexEntry.SetText(data.SysEx)
	mw.midiSysexEntry.OnChanged = func(s string) { mw.updateMidiJSON() }

	// Add Components
	mw.actionEditorContent.Add(container.NewBorder(nil, nil, widget.NewLabel("Device:"), nil, mw.midiDeviceSelect))
	mw.actionEditorContent.Add(mw.midiMsgTypeSelect)

	// Create param container but don't add all children yet - handled by updateMidiEditorVisibility which adds/removes?
	// Actually Fyne containers are static lists. We need to rebuild the param form or have hidden fields.
	// Let's just add all fields to a container and hide/show them?
	// Fyne widgets have Hide/Show but they still take layout space in some containers.
	// VBox handles Hide() correctly by collapsing space.

	// Re-trigger visibility update to ensure correct initial state
	mw.updateMidiEditorVisibility(displayType)

	// Add everything, visibility controlled by updateMidiEditorVisibility
	mw.actionEditorContent.Add(mw.midiChannelSelect) // Wrapped in container with specific label in visibility? No, just raw widget here is odd.

	// Helper to make labelled rows
	row := func(label string, w fyne.CanvasObject) *fyne.Container {
		return container.NewBorder(nil, nil, widget.NewLabel(label), nil, w)
	}

	mw.actionEditorContent.Add(mw.midiChannelRow(row))
	mw.actionEditorContent.Add(mw.midiNoteRow(row))
	mw.actionEditorContent.Add(mw.midiVelocityRow(row))
	mw.actionEditorContent.Add(mw.midiProgramRow(row))
	mw.actionEditorContent.Add(mw.midiSysexRow(row))
}

func (mw *MainWindow) midiChannelRow(rowFunc func(string, fyne.CanvasObject) *fyne.Container) *fyne.Container {
	return rowFunc("Channel:", mw.midiChannelSelect)
}
func (mw *MainWindow) midiNoteRow(rowFunc func(string, fyne.CanvasObject) *fyne.Container) *fyne.Container {
	return rowFunc("Number/Note:", mw.midiNoteEntry)
}
func (mw *MainWindow) midiVelocityRow(rowFunc func(string, fyne.CanvasObject) *fyne.Container) *fyne.Container {
	return rowFunc("Value/Velocity:", mw.midiVelocityEntry)
}
func (mw *MainWindow) midiProgramRow(rowFunc func(string, fyne.CanvasObject) *fyne.Container) *fyne.Container {
	return rowFunc("Program:", mw.midiProgramEntry)
}
func (mw *MainWindow) midiSysexRow(rowFunc func(string, fyne.CanvasObject) *fyne.Container) *fyne.Container {
	return rowFunc("Bytes:", mw.midiSysexEntry)
}

func (mw *MainWindow) updateMidiEditorVisibility(displayType string) {
	// Helper to find parent container of widget would be nice, but we don't have back references easily.
	// Instead, we just show/hide the widgets themselves. The row containers in VBox *should* collapse.
	// NOTE: If the row container is what's added to actionEditorContent, we need to hide the ROW container.
	// Implementing this cleanly: we should store the Row Containers as struct members or recreate them.
	// Let's rely on traversing actionEditorContent.Objects? No, unsafe.

	// Quick fix: assume references are available via widget parenting... no.
	// Let's access the widgets and hide/show them. If they are in a Border layout (row),
	// hiding the content usually makes the border layout collapse if configured right, or we hide the border container?
	// We need to hide the Border Container!

	// Since I didn't store the Border Containers, I'll need to reconstruct the editor content on Type change.
	// But `updateMidiEditorVisibility` is called from RadioGroup callback.

	// Better approach: Rebuild `actionEditorContent` entirely when MIDI type changes.

	// Refreshing the whole editor content container with just the fields we need.

	// Clear all params first (keep device and type selector)
	// We need to preserve the Device and Type selectors at the top.
	objs := mw.actionEditorContent.Objects
	if len(objs) < 2 {
		return
	} // Not fully built yet

	// Keep the first 2 (Device, Type)
	baseObjs := objs[:2]
	mw.actionEditorContent.Objects = baseObjs

	row := func(label string, w fyne.CanvasObject) *fyne.Container {
		return container.NewBorder(nil, nil, widget.NewLabel(label), nil, w)
	}

	switch displayType {
	case "Note On", "Note Off", "CC":
		mw.actionEditorContent.Add(row("Channel:", mw.midiChannelSelect))
		mw.actionEditorContent.Add(row("Number/Note:", mw.midiNoteEntry))
		mw.actionEditorContent.Add(row("Value/Velocity:", mw.midiVelocityEntry))
	case "PC":
		mw.actionEditorContent.Add(row("Channel:", mw.midiChannelSelect))
		mw.actionEditorContent.Add(row("Program:", mw.midiProgramEntry))
	case "SysEx":
		mw.actionEditorContent.Add(row("Bytes:", mw.midiSysexEntry))
	}

	mw.actionEditorContent.Refresh()
}

func (mw *MainWindow) updateMidiJSON() {
	if mw.selectedAction == nil || mw.selectedAction.Type != actions.ActionTypeMidi {
		return
	}

	// Gather data
	data := actions.MidiActionData{
		DeviceName: mw.midiDeviceSelect.Selected,
		Channel:    1,
		Note:       0,
		Velocity:   0,
		Program:    0,
		SysEx:      mw.midiSysexEntry.Text,
	}

	// Parse Type
	switch mw.midiMsgTypeSelect.Selected {
	case "Note On":
		data.MsgType = "note_on"
	case "Note Off":
		data.MsgType = "note_off"
	case "CC":
		data.MsgType = "cc"
	case "PC":
		data.MsgType = "pc"
	case "SysEx":
		data.MsgType = "sysex"
	}

	// Parse Ints
	if c, err := strconv.Atoi(mw.midiChannelSelect.Selected); err == nil {
		data.Channel = c
	}
	if n, err := strconv.Atoi(mw.midiNoteEntry.Text); err == nil {
		data.Note = n
	}
	if v, err := strconv.Atoi(mw.midiVelocityEntry.Text); err == nil {
		data.Velocity = v
	}
	if p, err := strconv.Atoi(mw.midiProgramEntry.Text); err == nil {
		data.Program = p
	}

	// Serialize
	bytes, _ := json.Marshal(data)
	mw.selectedAction.Code = string(bytes)
	mw.actionStore.UpdateAction(mw.selectedAction)
}

func (mw *MainWindow) addActionGroup() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Group Name")
	entry.SetText("New Group")

	dialog.ShowCustomConfirm("Create Action Group", "Create", "Cancel",
		container.NewVBox(widget.NewLabel("Enter a name for the group:"), entry),
		func(confirm bool) {
			if confirm && entry.Text != "" {
				group := actions.NewActionGroup(entry.Text)
				// If a group is selected, add as child of that group
				if mw.selectedGroup != nil {
					group.ParentGroupID = mw.selectedGroup.ID
				}
				mw.actionStore.AddGroup(group)
				mw.actionList.Refresh()
			}
		}, mw.window)
}

func (mw *MainWindow) addAction() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Action Name")
	entry.SetText("New Action")

	dialog.ShowCustomConfirm("Create Action", "Create", "Cancel",
		container.NewVBox(widget.NewLabel("Enter a name for the action:"), entry),
		func(confirm bool) {
			if confirm && entry.Text != "" {
				action := actions.NewAction(entry.Text, actions.ActionTypeShellCommand)
				action.WaitForCompletion = true // Default to true
				// If a group is selected, add as child of that group
				if mw.selectedGroup != nil {
					action.ParentGroupID = mw.selectedGroup.ID
				}
				mw.actionStore.AddAction(action)
				mw.actionList.Refresh()
			}
		}, mw.window)
}

func (mw *MainWindow) deleteSelectedActionItem() {
	if mw.selectedAction != nil {
		dialog.ShowConfirm("Delete Action", "Are you sure you want to delete '"+mw.selectedAction.Name+"'?",
			func(confirm bool) {
				if confirm {
					mw.actionStore.RemoveAction(mw.selectedAction.ID)
					mw.selectedAction = nil
					mw.actionList.Refresh()
					mw.updateActionEditor()
				}
			}, mw.window)
	} else if mw.selectedGroup != nil {
		dialog.ShowConfirm("Delete Group", "Are you sure you want to delete '"+mw.selectedGroup.Name+"' and all its contents?",
			func(confirm bool) {
				if confirm {
					mw.actionStore.RemoveGroup(mw.selectedGroup.ID)
					mw.selectedGroup = nil
					mw.actionList.Refresh()
					mw.updateActionEditor()
				}
			}, mw.window)
	}
}

func (mw *MainWindow) moveSelectedActionUp() {
	if mw.selectedAction != nil {
		mw.actionStore.MoveActionUp(mw.selectedAction.ID)
		mw.actionList.Refresh()
	} else if mw.selectedGroup != nil {
		mw.actionStore.MoveGroupUp(mw.selectedGroup.ID)
		mw.actionList.Refresh()
	}
}

func (mw *MainWindow) moveSelectedActionDown() {
	if mw.selectedAction != nil {
		mw.actionStore.MoveActionDown(mw.selectedAction.ID)
		mw.actionList.Refresh()
	} else if mw.selectedGroup != nil {
		mw.actionStore.MoveGroupDown(mw.selectedGroup.ID)
		mw.actionList.Refresh()
	}
}

func (mw *MainWindow) testAction() {
	if mw.selectedAction == nil {
		mw.actionFeedback.SetText("No action selected")
		return
	}

	mw.actionFeedback.SetText("Running...")

	// Use RunAction instead of direct Executor call to test async/group logic?
	// The RunAction method is simpler, but for "Test" button usually we want feedback.
	// Let's keep direct execution for Feedback, but maybe respect WaitForCompletion/Sleep?
	// Sleep already blocks, so running in goroutine is good.

	go func() {
		output, err := mw.executor.Execute(mw.selectedAction)
		if err != nil {
			mw.actionFeedback.SetText("Error: " + err.Error())
		} else if output != "" {
			mw.actionFeedback.SetText("Output: " + output)
		} else {
			mw.actionFeedback.SetText("Success (no output)")
		}
	}()
}

func (mw *MainWindow) validateAction() {
	if mw.selectedAction == nil {
		mw.actionFeedback.SetText("No action selected")
		return
	}

	var err error
	switch mw.selectedAction.Type {
	case actions.ActionTypeAppleScript:
		err = mw.executor.ValidateAppleScript(mw.selectedAction.Code)
	case actions.ActionTypeShellCommand:
		err = mw.executor.ValidateShellCommand(mw.selectedAction.Code)
	case actions.ActionTypeSleep:
		// Basic number check
		_, e := strconv.ParseFloat(mw.selectedAction.Code, 64)
		err = e
	case actions.ActionTypeMidi:
		// Check JSON validity
		var data actions.MidiActionData
		err = json.Unmarshal([]byte(mw.selectedAction.Code), &data)
	}

	if err != nil {
		mw.actionFeedback.SetText("Validation error: " + err.Error())
	} else {
		mw.actionFeedback.SetText("✓ Valid syntax")
	}
}

func (mw *MainWindow) saveActions() {
	mw.cfg.SyncActionStore(mw.actionStore)
	if err := mw.cfg.Save(); err != nil {
		log.Printf("Failed to save actions: %v", err)
		dialog.ShowError(err, mw.window)
	} else {
		// Refresh the action dropdown in the Menu Editor
		mw.refreshPadActionOptions()
		dialog.ShowInformation("Saved", "Actions saved successfully.", mw.window)
	}
}
