package islands

import (
	"fmt"

	"rogchap.com/v8go"
)

var iso = v8go.NewIsolate()

type Element struct {
	id         string
	entryPoint string
	marker     string
	Props      map[string]any
	CSR        bool
	SSR        bool
}

type Ctx struct {
	ResolveDir string
	Elements   map[string]*Element
}

type Framework struct {
	importMap           map[string]string
	jsxImportSource     string
	createRenderScript  func(ctx *Ctx) (string, error)
	createHydrateScript func(ctx *Ctx) (string, error)
}

func NewContext(resolveDir string) Ctx {
	return Ctx{
		ResolveDir: resolveDir,
		Elements:   map[string]*Element{},
	}
}

func (ctx *Ctx) needsSSR() bool {
	for _, el := range ctx.Elements {
		if el.SSR {
			return true
		}
	}
	return false
}

func (ctx *Ctx) needsCSR() bool {
	for _, el := range ctx.Elements {
		if el.CSR {
			return true
		}
	}
	return false
}

func (ctx *Ctx) AddElement(entryPoint string, props map[string]any) *Element {
	num := len(ctx.Elements)
	id := fmt.Sprintf("$h%d", num)
	marker := fmt.Sprintf("<!-- %s -->", id)
	element := &Element{
		id:         id,
		entryPoint: entryPoint,
		marker:     marker,
		Props:      props,
		SSR:        true,
		CSR:        false,
	}
	ctx.Elements[id] = element
	return element
}

func (e *Element) String() string {
	return fmt.Sprintf(`<div id="%s">%s</div>`, e.id, e.marker)
}
