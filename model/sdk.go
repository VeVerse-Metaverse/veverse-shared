package model

import "fmt"

type SDK struct {
	Entity
	Releases *ReleaseV2Batch `json:"releases,omitempty"` // list of releases for the SDK (required)
}

func (s *SDK) String() string {
	var out = s.Entity.String()
	if s.Releases != nil {
		out += fmt.Sprintf("releases:\n\t%v, ", (*Batch[ReleaseV2])(s.Releases).String())
	}
	return out
}
