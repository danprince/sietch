package mdext

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type syntaxHighlighting struct {
	style   string
	options []html.Option
}

func (e *syntaxHighlighting) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(e, 200),
	))
}

func NewSyntaxHighlighting(style string, options ...html.Option) *syntaxHighlighting {
	return &syntaxHighlighting{style: style, options: options}
}

func (r *syntaxHighlighting) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

func (r *syntaxHighlighting) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)

	if !entering {
		return ast.WalkContinue, nil
	}

	language, highlights := parseHighlightRanges(string(n.Language(source)))

	lexer := lexers.Get(string(language))

	if lexer == nil {
		lexer = lexers.Fallback
	}

	theme := r.style

	if r.style == "css" {
		theme = "github"
	}

	lexer = chroma.Coalesce(lexer)
	style := styles.Get(theme)

	options := []html.Option{
		html.Standalone(false),
		html.HighlightLines(highlights),
		html.WithClasses(r.style == "css"),
	}

	options = append(options, r.options...)
	formatter := html.New(options...)

	var buffer bytes.Buffer
	lines := n.Lines()
	linesLen := lines.Len()
	for i := 0; i < linesLen; i++ {
		line := lines.At(i)
		buffer.Write(line.Value(source))
	}

	iterator, err := lexer.Tokenise(nil, buffer.String())

	if err != nil {
		return ast.WalkStop, err
	}

	formatter.Format(w, style, iterator)

	return ast.WalkContinue, nil
}

type lineRange = [2]int

// Parses highlight line ranges from a fenced codeblock language name in
// the prismjs format: https://prismjs.com/plugins/line-highlight/#how-to-use
func parseHighlightRanges(s string) (string, []lineRange) {
	offset := strings.IndexByte(s, '/')

	if offset < 0 {
		return s, []lineRange{}
	}

	lang := s[:offset]
	rest := s[offset+1:]
	rest = strings.ReplaceAll(rest, " ", "")
	parts := strings.Split(rest, ",")
	ranges := make([]lineRange, 0, len(parts))

	for _, part := range parts {
		lineNums := strings.Split(part, "-")
		start := -1
		end := -1

		if len(lineNums) >= 1 {
			start, _ = strconv.Atoi(lineNums[0])
		}

		if len(lineNums) >= 2 {
			end, _ = strconv.Atoi(lineNums[1])
		}

		if end < start {
			end = start
		}

		// Chroma wants zero based indexes
		start -= 1
		end -= 1

		if start >= 0 {
			ranges = append(ranges, lineRange{start, end})
		}
	}

	return lang, ranges
}
