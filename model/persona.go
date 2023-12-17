package model

import "fmt"

type Persona struct {
	Entity

	Name string `json:"name,omitempty"`
}

func (p *Persona) String() string {
	var out = ""
	out += fmt.Sprintf("id: %v, ", p.Id)
	out += fmt.Sprintf("entityType: %v, ", p.EntityType)
	out += fmt.Sprintf("public: %v, ", p.Public)
	out += fmt.Sprintf("views: %v, ", p.Views)
	out += fmt.Sprintf("owner: %v, ", p.Owner.String())
	out += fmt.Sprintf("name: %v, ", p.Name)
	return out
}
