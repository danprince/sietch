package mdext

import (
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type headingAnchors struct {
}

func (h *headingAnchors) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithAutoHeadingID(),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(util.Prioritized(h, 200)),
	)
}

// Wraps each heading in an anchor tag, linking to itself
var HeadingAnchors = &headingAnchors{}

func (h *headingAnchors) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, h.renderHeading)
}

func (h *headingAnchors) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	name := fmt.Sprintf("h%d", n.Level)
	id, _ := node.Attribute([]byte("id"))
	autolink := n.FirstChild().Kind() != ast.KindLink

	if autolink && entering {
		w.WriteString(fmt.Sprintf(`<a href="#%s" class="permalink">`, id))
		w.WriteString(fmt.Sprintf(`<%s id="%s">`, name, id))
	} else if autolink {
		w.WriteString(fmt.Sprintf("</%s>", name))
		w.WriteString("</a>")
	} else if entering {
		w.WriteString(fmt.Sprintf(`<%s>`, name))
	} else {
		w.WriteString(fmt.Sprintf("</%s>", name))
	}

	return ast.WalkContinue, nil
}
