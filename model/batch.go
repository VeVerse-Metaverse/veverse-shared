package model

import "fmt"

type Batch[T interface{}] struct {
	Entities []T    `json:"entities,omitempty"`
	Offset   int64  `json:"offset,omitempty"`
	Limit    int64  `json:"limit,omitempty"`
	Total    uint64 `json:"total,omitempty"`
}

func (b *Batch[T]) String() string {
	var out = ""
	out += fmt.Sprintf("offset: %v, ", b.Offset)
	out += fmt.Sprintf("limit: %v, ", b.Limit)
	out += fmt.Sprintf("total: %v, ", b.Total)
	out += "entities: [ "
	for _, v := range b.Entities {
		if s, ok := any(v).(fmt.Stringer); ok {
			out += fmt.Sprintf("\n\t\t%v, ", s.String())
		} else {
			out += fmt.Sprintf("\n\t\t%v, ", v)
		}
	}
	out += "], "
	return out
}
