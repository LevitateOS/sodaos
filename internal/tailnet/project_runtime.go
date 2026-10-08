package tailnet

// ProjectSelection is reviewed explicitly by the creating human. Omission or
// Enabled=false means Off regardless of a subsequently changed operator default.
type ProjectSelection struct {
	Enabled  bool   `json:"enabled"`
	Revision string `json:"revision,omitempty"`
	Binding  string `json:"binding,omitempty"`
}

func (s ProjectSelection) Validate() error {
	if !s.Enabled {
		if s.Revision != "" || s.Binding != "" {
			return ErrInvalid
		}
	} else if !revisionPattern.MatchString(s.Revision) || !revisionPattern.MatchString(s.Binding) {
		return ErrInvalid
	}
	return nil
}
