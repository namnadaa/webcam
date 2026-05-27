package ui

import "time"

// State stores visibility flags for UI panels.
type State struct {
	ShowMenu     bool
	ShowStats    bool
	ShowSettings bool
	ShowStatuses bool
	ShowControls bool
	Notification Notification
}

// Notification stores temporary UI notification data.
type Notification struct {
	Message string
	Until   time.Time
}

// ShowNotification displays a notification for a limited time.
func (s *State) ShowNotification(msg string, duration time.Duration) {
	s.Notification.Message = msg
	s.Notification.Until = time.Now().Add(duration)
}

// NewUIState creates default UI state.
func NewUIState() State {
	return State{
		ShowMenu:     true,
		ShowStats:    false,
		ShowSettings: false,
		ShowStatuses: false,
		ShowControls: false,
	}
}
