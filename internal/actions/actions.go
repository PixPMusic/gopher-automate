package actions

import (
	"sort"

	"github.com/google/uuid"
)

// ActionType represents the type of action to execute
type ActionType string

const (
	ActionTypeAppleScript  ActionType = "applescript"
	ActionTypeShellCommand ActionType = "shell"
	ActionTypeSleep        ActionType = "sleep"
	ActionTypeMidi         ActionType = "midi"
)

// Action represents an executable action
type Action struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Type              ActionType `json:"type"`
	Code              string     `json:"code"`
	ParentGroupID     string     `json:"parent_group_id"`     // Empty if root-level
	Order             int        `json:"order"`               // For sorting within parent
	WaitForCompletion bool       `json:"wait_for_completion"` // Block next action until this one finishes
}

// ActionGroup is a named folder containing actions and other groups
type ActionGroup struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ParentGroupID string `json:"parent_group_id"` // Allows nested groups, empty if root-level
	Order         int    `json:"order"`           // For sorting within parent
}

// NewAction creates a new action with a generated ID
func NewAction(name string, actionType ActionType) *Action {
	return &Action{
		ID:   uuid.New().String(),
		Name: name,
		Type: actionType,
	}
}

// NewActionGroup creates a new action group with a generated ID
func NewActionGroup(name string) *ActionGroup {
	return &ActionGroup{
		ID:   uuid.New().String(),
		Name: name,
	}
}

// ActionStore manages actions and groups, providing tree operations
type ActionStore struct {
	Actions []Action
	Groups  []ActionGroup
}

// NewActionStore creates an empty action store
func NewActionStore() *ActionStore {
	return &ActionStore{
		Actions: []Action{},
		Groups:  []ActionGroup{},
	}
}

// AddAction adds an action to the store
func (s *ActionStore) AddAction(action *Action) {
	// Set order to be last in its parent
	action.Order = s.getNextOrder(action.ParentGroupID)
	s.Actions = append(s.Actions, *action)
}

// AddGroup adds a group to the store
func (s *ActionStore) AddGroup(group *ActionGroup) {
	group.Order = s.getNextOrder(group.ParentGroupID)
	s.Groups = append(s.Groups, *group)
}

// getNextOrder returns the next order value for items in a parent
func (s *ActionStore) getNextOrder(parentID string) int {
	maxOrder := -1
	for _, a := range s.Actions {
		if a.ParentGroupID == parentID && a.Order > maxOrder {
			maxOrder = a.Order
		}
	}
	for _, g := range s.Groups {
		if g.ParentGroupID == parentID && g.Order > maxOrder {
			maxOrder = g.Order
		}
	}
	return maxOrder + 1
}

// GetAction returns an action by ID, or nil if not found
func (s *ActionStore) GetAction(id string) *Action {
	for i := range s.Actions {
		if s.Actions[i].ID == id {
			return &s.Actions[i]
		}
	}
	return nil
}

// GetGroup returns a group by ID, or nil if not found
func (s *ActionStore) GetGroup(id string) *ActionGroup {
	for i := range s.Groups {
		if s.Groups[i].ID == id {
			return &s.Groups[i]
		}
	}
	return nil
}

// UpdateAction updates an existing action
func (s *ActionStore) UpdateAction(action *Action) bool {
	for i := range s.Actions {
		if s.Actions[i].ID == action.ID {
			s.Actions[i] = *action
			return true
		}
	}
	return false
}

// UpdateGroup updates an existing group
func (s *ActionStore) UpdateGroup(group *ActionGroup) bool {
	for i := range s.Groups {
		if s.Groups[i].ID == group.ID {
			s.Groups[i] = *group
			return true
		}
	}
	return false
}

// RemoveAction removes an action by ID
func (s *ActionStore) RemoveAction(id string) bool {
	for i := range s.Actions {
		if s.Actions[i].ID == id {
			s.Actions = append(s.Actions[:i], s.Actions[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveGroup removes a group by ID and all its children (recursively)
func (s *ActionStore) RemoveGroup(id string) bool {
	// First, recursively remove all children
	s.removeChildrenOfGroup(id)

	// Then remove the group itself
	for i := range s.Groups {
		if s.Groups[i].ID == id {
			s.Groups = append(s.Groups[:i], s.Groups[i+1:]...)
			return true
		}
	}
	return false
}

// removeChildrenOfGroup removes all actions and groups that are children of the given group
func (s *ActionStore) removeChildrenOfGroup(parentID string) {
	// Find child groups first
	childGroupIDs := []string{}
	for _, g := range s.Groups {
		if g.ParentGroupID == parentID {
			childGroupIDs = append(childGroupIDs, g.ID)
		}
	}

	// Recursively remove children of child groups
	for _, childID := range childGroupIDs {
		s.removeChildrenOfGroup(childID)
	}

	// Remove actions in this group
	newActions := []Action{}
	for _, a := range s.Actions {
		if a.ParentGroupID != parentID {
			newActions = append(newActions, a)
		}
	}
	s.Actions = newActions

	// Remove child groups
	newGroups := []ActionGroup{}
	for _, g := range s.Groups {
		if g.ParentGroupID != parentID {
			newGroups = append(newGroups, g)
		}
	}
	s.Groups = newGroups
}

// MoveAction moves an action to a new parent group with a specific order
func (s *ActionStore) MoveAction(actionID, newParentID string, newOrder int) bool {
	action := s.GetAction(actionID)
	if action == nil {
		return false
	}

	oldParentID := action.ParentGroupID
	oldOrder := action.Order

	// Update the action
	action.ParentGroupID = newParentID
	action.Order = newOrder

	// Shift orders of other items
	s.shiftOrdersAfterRemove(oldParentID, oldOrder)
	s.shiftOrdersAfterInsert(newParentID, newOrder, actionID)

	return s.UpdateAction(action)
}

// MoveGroup moves a group to a new parent group with a specific order
func (s *ActionStore) MoveGroup(groupID, newParentID string, newOrder int) bool {
	group := s.GetGroup(groupID)
	if group == nil {
		return false
	}

	// Prevent moving a group into itself or its descendants
	if s.isDescendantOf(newParentID, groupID) {
		return false
	}

	oldParentID := group.ParentGroupID
	oldOrder := group.Order

	// Update the group
	group.ParentGroupID = newParentID
	group.Order = newOrder

	// Shift orders
	s.shiftOrdersAfterRemove(oldParentID, oldOrder)
	s.shiftOrdersAfterInsert(newParentID, newOrder, groupID)

	return s.UpdateGroup(group)
}

// MoveActionUp moves an action up in its parent (decreases order)
func (s *ActionStore) MoveActionUp(actionID string) bool {
	action := s.GetAction(actionID)
	if action == nil || action.Order == 0 {
		return false
	}

	// Find sibling with order = action.Order - 1
	for i := range s.Actions {
		if s.Actions[i].ParentGroupID == action.ParentGroupID && s.Actions[i].Order == action.Order-1 {
			s.Actions[i].Order++
			action.Order--
			return s.UpdateAction(action)
		}
	}
	for i := range s.Groups {
		if s.Groups[i].ParentGroupID == action.ParentGroupID && s.Groups[i].Order == action.Order-1 {
			s.Groups[i].Order++
			action.Order--
			return s.UpdateAction(action)
		}
	}
	return false
}

// MoveActionDown moves an action down in its parent (increases order)
func (s *ActionStore) MoveActionDown(actionID string) bool {
	action := s.GetAction(actionID)
	if action == nil {
		return false
	}

	maxOrder := s.getMaxOrder(action.ParentGroupID)
	if action.Order >= maxOrder {
		return false
	}

	// Find sibling with order = action.Order + 1
	for i := range s.Actions {
		if s.Actions[i].ParentGroupID == action.ParentGroupID && s.Actions[i].Order == action.Order+1 {
			s.Actions[i].Order--
			action.Order++
			return s.UpdateAction(action)
		}
	}
	for i := range s.Groups {
		if s.Groups[i].ParentGroupID == action.ParentGroupID && s.Groups[i].Order == action.Order+1 {
			s.Groups[i].Order--
			action.Order++
			return s.UpdateAction(action)
		}
	}
	return false
}

// MoveGroupUp moves a group up in its parent (decreases order)
func (s *ActionStore) MoveGroupUp(groupID string) bool {
	group := s.GetGroup(groupID)
	if group == nil || group.Order == 0 {
		return false
	}

	// Find sibling with order = group.Order - 1
	for i := range s.Actions {
		if s.Actions[i].ParentGroupID == group.ParentGroupID && s.Actions[i].Order == group.Order-1 {
			s.Actions[i].Order++
			group.Order--
			return s.UpdateGroup(group)
		}
	}
	for i := range s.Groups {
		if s.Groups[i].ParentGroupID == group.ParentGroupID && s.Groups[i].Order == group.Order-1 {
			s.Groups[i].Order++
			group.Order--
			return s.UpdateGroup(group)
		}
	}
	return false
}

// MoveGroupDown moves a group down in its parent (increases order)
func (s *ActionStore) MoveGroupDown(groupID string) bool {
	group := s.GetGroup(groupID)
	if group == nil {
		return false
	}

	maxOrder := s.getMaxOrder(group.ParentGroupID)
	if group.Order >= maxOrder {
		return false
	}

	// Find sibling with order = group.Order + 1
	for i := range s.Actions {
		if s.Actions[i].ParentGroupID == group.ParentGroupID && s.Actions[i].Order == group.Order+1 {
			s.Actions[i].Order--
			group.Order++
			return s.UpdateGroup(group)
		}
	}
	for i := range s.Groups {
		if s.Groups[i].ParentGroupID == group.ParentGroupID && s.Groups[i].Order == group.Order+1 {
			s.Groups[i].Order--
			group.Order++
			return s.UpdateGroup(group)
		}
	}
	return false
}

// getMaxOrder returns the maximum order value for items in a parent
func (s *ActionStore) getMaxOrder(parentID string) int {
	maxOrder := -1
	for _, a := range s.Actions {
		if a.ParentGroupID == parentID && a.Order > maxOrder {
			maxOrder = a.Order
		}
	}
	for _, g := range s.Groups {
		if g.ParentGroupID == parentID && g.Order > maxOrder {
			maxOrder = g.Order
		}
	}
	return maxOrder
}

// isDescendantOf checks if potentialDescendant is a descendant of ancestorID
func (s *ActionStore) isDescendantOf(potentialDescendantID, ancestorID string) bool {
	if potentialDescendantID == "" {
		return false
	}
	if potentialDescendantID == ancestorID {
		return true
	}
	group := s.GetGroup(potentialDescendantID)
	if group == nil {
		return false
	}
	return s.isDescendantOf(group.ParentGroupID, ancestorID)
}

// shiftOrdersAfterRemove decrements orders of items after a removed item
func (s *ActionStore) shiftOrdersAfterRemove(parentID string, removedOrder int) {
	for i := range s.Actions {
		if s.Actions[i].ParentGroupID == parentID && s.Actions[i].Order > removedOrder {
			s.Actions[i].Order--
		}
	}
	for i := range s.Groups {
		if s.Groups[i].ParentGroupID == parentID && s.Groups[i].Order > removedOrder {
			s.Groups[i].Order--
		}
	}
}

// shiftOrdersAfterInsert increments orders of items at or after an inserted item
func (s *ActionStore) shiftOrdersAfterInsert(parentID string, insertedOrder int, excludeID string) {
	for i := range s.Actions {
		if s.Actions[i].ParentGroupID == parentID && s.Actions[i].Order >= insertedOrder && s.Actions[i].ID != excludeID {
			s.Actions[i].Order++
		}
	}
	for i := range s.Groups {
		if s.Groups[i].ParentGroupID == parentID && s.Groups[i].Order >= insertedOrder && s.Groups[i].ID != excludeID {
			s.Groups[i].Order++
		}
	}
}

// TreeItem represents a node in the action tree (either a group or action)
type TreeItem struct {
	IsGroup  bool
	Action   *Action
	Group    *ActionGroup
	Depth    int
	HasIcon  bool
	Expanded bool
}

// GetSortedTree returns a flat list representing the sorted tree of actions and groups
// Groups come before actions within the same parent, both sorted by Order
func (s *ActionStore) GetSortedTree(parentID string, depth int) []TreeItem {
	var items []TreeItem

	// Get groups in this parent
	var groups []ActionGroup
	for _, g := range s.Groups {
		if g.ParentGroupID == parentID {
			groups = append(groups, g)
		}
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Order < groups[j].Order
	})

	// Get actions in this parent
	var actions []Action
	for _, a := range s.Actions {
		if a.ParentGroupID == parentID {
			actions = append(actions, a)
		}
	}
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Order < actions[j].Order
	})

	// Add groups first, then recursively their children
	for _, g := range groups {
		gCopy := g
		items = append(items, TreeItem{
			IsGroup: true,
			Group:   &gCopy,
			Depth:   depth,
		})
		// Recursively add children
		items = append(items, s.GetSortedTree(g.ID, depth+1)...)
	}

	// Add actions
	for _, a := range actions {
		aCopy := a
		items = append(items, TreeItem{
			IsGroup: false,
			Action:  &aCopy,
			Depth:   depth,
		})
	}

	return items
}

// GetFlatList returns all actions in sorted tree order (for dropdowns)
func (s *ActionStore) GetFlatList() []TreeItem {
	return s.GetSortedTree("", 0)
}
