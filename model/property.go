package model

import "fmt"

// Custom entity property trait
type Property struct {
	EntityTrait

	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (p *Property) String() string {
	var out = p.EntityTrait.String()
	out += fmt.Sprintf("type: %v, ", p.Type)
	out += fmt.Sprintf("name: %v, ", p.Name)
	out += fmt.Sprintf("value: %v, ", p.Value)
	return out
}

type PropertyBatch Batch[Property]

type InsertProperty struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}
