package media

// State holds UI media action flags.
type State struct {
	Screenshot bool
	Recording  bool
}

// NewMediaState creates default media state.
func NewMediaState() State {
	return State{
		Screenshot: false,
		Recording:  false,
	}
}
