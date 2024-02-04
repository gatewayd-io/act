package act

import "fmt"

type Signal struct {
	Name     string
	Metadata map[string]any
	Sync     bool
}

func (s *Signal) String() string {
	return fmt.Sprintf("%s(%v)", s.Name, s.Metadata)
}
