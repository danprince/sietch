package islands

import (
	"encoding/json"
	"fmt"
)

type HydrationType uint8

const (
	Static HydrationType = iota
	ClientOnLoad
	ClientOnVisible
	ClientOnIdle
)

type Island struct {
	Id         string
	Type       HydrationType
	Props      Props
	EntryPoint string
	ClientOnly bool
}

// Helper for templates that turns an island into HTML.
func (i *Island) String() string {
	if i.Type == Static {
		return i.Marker()
	} else if i.ClientOnly {
		return fmt.Sprintf(`<div id="%s"></div>`, i.Id)
	} else {
		return fmt.Sprintf(`<div id="%s">%s</div>`, i.Id, i.Marker())
	}
}

func (i *Island) Marker() string {
	return fmt.Sprintf("<!-- %s -->", i.Id)
}

// Distinct type for props that stringifies to JSON.
type Props map[string]any

func (p Props) String() string {
	data, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return string(data)
}
