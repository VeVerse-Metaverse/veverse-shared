package model

import "fmt"

type Link struct {
	Identifier

	Url  string  `json:"url"`
	Name *string `json:"name"`
}

func (l *Link) String() string {
	var out = l.Identifier.String()
	out += fmt.Sprintf("url: %v, ", l.Url)
	out += fmt.Sprintf("name: %v, ", l.Name)
	return out
}

type LinkBatch Batch[Link]
