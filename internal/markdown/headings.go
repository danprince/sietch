package markdown

import (
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Enables the autoHeadingId at the parser, then wraps each heading inside an anchor.
type headings struct {
}

func (h *headings) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithAutoHeadingID(),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(util.Prioritized(h, 200)),
	)
}

var Headings = &headings{}

func (h *headings) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, h.renderHeading)
}

func (h *headings) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	name := fmt.Sprintf("h%d", n.Level)

	if entering {
		id, _ := node.Attribute([]byte("id"))
		w.WriteString(fmt.Sprintf(`<a href="#%s" class="permalink">`, id))
		w.WriteString(fmt.Sprintf(`<%s id="%s">`, name, id))
		return ast.WalkContinue, nil
	} else {
		w.WriteString(fmt.Sprintf("</%s>", name))
		w.WriteString("</a>")
		return ast.WalkContinue, nil
	}
}
