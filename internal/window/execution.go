package window

import (
	"log"

	"github.com/PixPMusic/gopher-automate/internal/actions"
)

// runAction executes a single action or a group of actions
// If the action is a group, it executes children sequentially.
// If isAsync is true, it runs in a goroutine (unless it's a child of a group, where parent controls flow).
func (mw *MainWindow) runAction(action *actions.Action, isAsync bool) {
	if action == nil {
		return
	}

	task := func() {
		mw.executeRecursive(action)
	}

	if isAsync {
		go task()
	} else {
		task()
	}
}

// runGroup executes all children of a group sequentially
func (mw *MainWindow) runGroup(group *actions.ActionGroup) {
	if group == nil {
		return
	}

	// Get sorted children
	children := mw.actionStore.GetSortedTree(group.ID, 0)

	// Execute each child
	for _, child := range children {
		if child.IsGroup {
			mw.runGroup(child.Group)
		} else {
			mw.executeRecursive(child.Action)
		}
	}
}

// executeRecursive executes an action via the executor.
// If it has WaitForCompletion=true, it blocks until done.
// If not, it fires async and returns immediately (allowing the caller to proceed).
func (mw *MainWindow) executeRecursive(action *actions.Action) {
	// If it's stored as a group in the ActionStore (although Action struct doesn't have IsGroup flag,
	// the store distinguishes).
	// Wait, the Pad Mapping stores an Action ID. That ID could belong to an Action OR a Group.
	// But `cfg.GetAction(id)` only searches Actions list. `cfg.GetGroup(id)` searches Groups.
	// We need to check both if we want to allow assigning Groups to buttons.

	// However, `handlePadPress` currently calls `mw.cfg.GetAction`.
	// If the user wants to assign a Group to a button, `GetAction` will return nil.
	// We should check `GetGroup` as well.

	// Execute the action
	if action.WaitForCompletion {
		if _, err := mw.executor.Execute(action); err != nil {
			log.Printf("Action '%s' failed: %v", action.Name, err)
		}
	} else {
		// Fire and forget
		go func() {
			if _, err := mw.executor.Execute(action); err != nil {
				log.Printf("Action '%s' failed: %v", action.Name, err)
			}
		}()
	}
}

func (mw *MainWindow) resolveAndRun(id string) {
	// Try action
	if action := mw.actionStore.GetAction(id); action != nil {
		mw.runAction(action, true) // Run top level action async
		return
	}

	// Try group
	if group := mw.actionStore.GetGroup(id); group != nil {
		go mw.runGroup(group) // Run group (sequential) async
		return
	}
}
