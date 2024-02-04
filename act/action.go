package act

// ActionFunc is a function that can be used to run an action.
type ActionFunc func(data map[string]any) (bool, error)

type Action struct {
	Name     string
	Metadata map[string]any
	Sync     bool
	Run      ActionFunc
}

func (a *Action) String() string {
	return a.Name
}
