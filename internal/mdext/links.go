package mdext

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type links struct {
}

func (e *links) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithASTTransformers(
		util.Prioritized(e, 200),
	))
}

// Adds the appropriate attributes for opening external links in a new tab
// and without a referrer/opener.
var Links = &links{}

func (t *links) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || n.Kind() != ast.KindLink {
			return ast.WalkContinue, nil
		}

		link := n.(*ast.Link)
		src := string(link.Destination)

		if strings.HasPrefix(src, "http") || strings.HasPrefix(src, "://") {
			link.SetAttribute([]byte("target"), []byte("_blank"))
			link.SetAttribute([]byte("rel"), []byte("noopener noreferrer"))
		}

		if strings.HasSuffix(src, ".md") {
			src = strings.Replace(src, ".md", ".html", 1)
			src = strings.Replace(src, "index.html", "", 1)
			link.Destination = []byte(src)
		}

		return ast.WalkContinue, nil
	})
}
