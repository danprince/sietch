package mdext

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type externalLinks struct {
}

func (e *externalLinks) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithASTTransformers(
		util.Prioritized(e, 200),
	))
}

// Adds the appropriate attributes for opening external links in a new tab
// and without a referrer/opener.
var ExternalLinks = &externalLinks{}

func (t *externalLinks) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
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

		return ast.WalkContinue, nil
	})
}
