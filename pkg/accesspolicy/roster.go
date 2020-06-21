package user

import "sync"

// RightsRoster denotes who has what rights
type RightsRoster struct {
	Everyone AccessRight            `json:"everyone"`
	Role     map[uint32]AccessRight `json:"role"`
	Group    map[uint32]AccessRight `json:"group"`
	User     map[uint32]AccessRight `json:"user"`

	changes []accessChange
	sync.RWMutex
}

// NewRightsRoster is a shorthand initializer function
func NewRightsRoster() *RightsRoster {
	return &RightsRoster{
		Everyone: APNoAccess,
		Group:    make(map[uint32]AccessRight),
		Role:     make(map[uint32]AccessRight),
		User:     make(map[uint32]AccessRight),
	}
}

// addChange adds a single change for further storing
func (rr *RightsRoster) addChange(action RRAction, subjectKind SubjectKind, subjectID uint32, rights AccessRight) {
	change := accessChange{
		action:      action,
		subjectKind: subjectKind,
		subjectID:   subjectID,
		accessRight: rights,
	}

	rr.Lock()

	if rr.changes == nil {
		rr.changes = []accessChange{change}
	} else {
		rr.changes = append(rr.changes, change)
	}

	rr.Unlock()
}

func (rr *RightsRoster) clearChanges() {
	rr.Lock()
	rr.changes = nil
	rr.Unlock()
}

// Summarize summarizing the resulting access right flags
func (rr *RightsRoster) Summarize(ctx context.Context, userID uint32) AccessRight {
	gm := ctx.Value(CKGroupManager).(*GroupManager)
	if gm == nil {
		panic(ErrNilGroupManager)
	}

	r := rr.Everyone

	// calculating standard and role group rights
	// NOTE: if some group doesn't have explicitly set rights, then
	// attempting to obtain the rights of a first ancestor group,
	// that has specific rights set
	for _, g := range gm.GroupsByUserID(ctx, userID, GKRole|GKGroup) {
		r |= rr.GroupRights(ctx, g.ID)
	}

	// user-specific rights
	if _, ok := rr.User[userID]; ok {
		r |= rr.User[userID]
	}

	return r
}

// GroupRights returns the rights of a given group if set explicitly,
// otherwise returns the rights of the first ancestor group that has
// any rights record explicitly set
func (rr *RightsRoster) GroupRights(ctx context.Context, groupID uint32) AccessRight {
	if groupID == 0 {
		return APNoAccess
	}

	var rights AccessRight
	var ok bool

	// obtaining group manager
	gm := ctx.Value(CKGroupManager).(*GroupManager)
	if gm == nil {
		panic(ErrNilGroupManager)
	}

	// obtaining target group
	g, err := gm.GroupByID(ctx, groupID)
	if err != nil {
		return APNoAccess
	}

	rr.RLock()

	switch g.Kind {
	case GKGroup:
		rights, ok = rr.Group[g.ID]
	case GKRole:
		rights, ok = rr.Role[g.ID]
	}

	rr.RUnlock()

	if ok {
		return rights
	}

	// now looking for the first set rights by tracing back
	// through its parents
	if g.ParentID != 0 {
		return rr.GroupRights(ctx, g.ParentID)
	}

	return APNoAccess
}
