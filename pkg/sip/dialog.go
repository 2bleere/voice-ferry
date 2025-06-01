package sip

import (
	"fmt"
	"sync"
	"time"

	"github.com/emiago/sipgo/sip"
)

// Dialog represents a SIP dialog
type Dialog struct {
	ID           string
	CallID       string
	LocalTag     string
	RemoteTag    string
	LocalURI     string
	RemoteURI    string
	RouteSet     []string
	State        DialogState
	CreatedAt    time.Time
	LastActivity time.Time
	IsUAS        bool    // true if this is the UAS side
	PeerDialog   *Dialog // reference to the other leg in B2BUA
}

// DialogState represents the state of a SIP dialog
type DialogState int

const (
	DialogStateEarly DialogState = iota
	DialogStateConfirmed
	DialogStateTerminated
)

func (s DialogState) String() string {
	switch s {
	case DialogStateEarly:
		return "Early"
	case DialogStateConfirmed:
		return "Confirmed"
	case DialogStateTerminated:
		return "Terminated"
	default:
		return "Unknown"
	}
}

// DialogManager manages SIP dialogs for the B2BUA
type DialogManager struct {
	mu      sync.RWMutex
	dialogs map[string]*Dialog
}

// NewDialogManager creates a new dialog manager
func NewDialogManager() *DialogManager {
	return &DialogManager{
		dialogs: make(map[string]*Dialog),
	}
}

// CreateDialog creates a new dialog from a SIP request
func (dm *DialogManager) CreateDialog(req *sip.Request, isUAS bool) *Dialog {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	var localTag, remoteTag string
	var localURI, remoteURI string

	if isUAS {
		// For UAS, we generate the local tag (To tag)
		localTag = generateTag()
		remoteTag = req.From().Params["tag"]
		localURI = req.To().Address.String()
		remoteURI = req.From().Address.String()
	} else {
		// For UAC, we generate the local tag (From tag)
		localTag = generateTag()
		if req.To().Params != nil {
			remoteTag = req.To().Params["tag"]
		}
		localURI = req.From().Address.String()
		remoteURI = req.To().Address.String()
	}

	callID := req.CallID().Value()
	dialogID := fmt.Sprintf("%s-%s-%s", callID, localTag, remoteTag)

	dialog := &Dialog{
		ID:           dialogID,
		CallID:       callID,
		LocalTag:     localTag,
		RemoteTag:    remoteTag,
		LocalURI:     localURI,
		RemoteURI:    remoteURI,
		State:        DialogStateEarly,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		IsUAS:        isUAS,
	}

	// Extract route set from Record-Route headers
	if recordRoute := req.RecordRoute(); recordRoute != nil {
		dialog.RouteSet = append(dialog.RouteSet, recordRoute.Address.String())
	}

	dm.dialogs[dialogID] = dialog
	return dialog
}

// FindDialog finds a dialog by Call-ID and tags
func (dm *DialogManager) FindDialog(callID, fromURI, toURI string) *Dialog {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	// Search through all dialogs for a match
	for _, dialog := range dm.dialogs {
		if dialog.CallID == callID {
			// For UAS dialog, remote is from, local is to
			if dialog.IsUAS && dialog.RemoteURI == fromURI && dialog.LocalURI == toURI {
				return dialog
			}
			// For UAC dialog, local is from, remote is to
			if !dialog.IsUAS && dialog.LocalURI == fromURI && dialog.RemoteURI == toURI {
				return dialog
			}
		}
	}

	return nil
}

// FindDialogByID finds a dialog by its ID
func (dm *DialogManager) FindDialogByID(dialogID string) *Dialog {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.dialogs[dialogID]
}

// UpdateDialogState updates the state of a dialog
func (dm *DialogManager) UpdateDialogState(dialogID string, state DialogState) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dialog, exists := dm.dialogs[dialogID]; exists {
		dialog.State = state
		dialog.LastActivity = time.Now()
	}
}

// ConfirmDialog moves a dialog from early to confirmed state
func (dm *DialogManager) ConfirmDialog(dialogID string, remoteTag string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dialog, exists := dm.dialogs[dialogID]; exists {
		if dialog.RemoteTag == "" {
			dialog.RemoteTag = remoteTag
			// Update dialog ID to include the remote tag
			delete(dm.dialogs, dialogID)
			newDialogID := fmt.Sprintf("%s-%s-%s", dialog.CallID, dialog.LocalTag, dialog.RemoteTag)
			dialog.ID = newDialogID
			dm.dialogs[newDialogID] = dialog
		}
		dialog.State = DialogStateConfirmed
		dialog.LastActivity = time.Now()
	}
}

// RemoveDialog removes a dialog from the manager
func (dm *DialogManager) RemoveDialog(dialogID string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dialog, exists := dm.dialogs[dialogID]; exists {
		dialog.State = DialogStateTerminated
		delete(dm.dialogs, dialogID)
	}
}

// LinkDialogs links two dialogs as B2BUA legs
func (dm *DialogManager) LinkDialogs(dialog1, dialog2 *Dialog) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dialog1.PeerDialog = dialog2
	dialog2.PeerDialog = dialog1
}

// GetActiveDialogs returns all active dialogs
func (dm *DialogManager) GetActiveDialogs() []*Dialog {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var active []*Dialog
	for _, dialog := range dm.dialogs {
		if dialog.State != DialogStateTerminated {
			active = append(active, dialog)
		}
	}

	return active
}

// GetDialogCount returns the total number of active dialogs
func (dm *DialogManager) GetDialogCount() int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	count := 0
	for _, dialog := range dm.dialogs {
		if dialog.State != DialogStateTerminated {
			count++
		}
	}

	return count
}

// CleanupExpiredDialogs removes dialogs that have been inactive for too long
func (dm *DialogManager) CleanupExpiredDialogs(maxAge time.Duration) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	now := time.Now()
	for dialogID, dialog := range dm.dialogs {
		if now.Sub(dialog.LastActivity) > maxAge {
			dialog.State = DialogStateTerminated
			delete(dm.dialogs, dialogID)
		}
	}
}

// generateTag generates a random tag for dialog identification
func generateTag() string {
	// Simple tag generation - in production, use crypto/rand
	return fmt.Sprintf("tag-%d", time.Now().UnixNano())
}

// UpdateActivity updates the last activity time for a dialog
func (dm *DialogManager) UpdateActivity(dialogID string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dialog, exists := dm.dialogs[dialogID]; exists {
		dialog.LastActivity = time.Now()
	}
}
