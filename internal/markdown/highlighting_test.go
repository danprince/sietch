package markdown

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	_ "embed"

	chromaHtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

func TestParseHighlightRanges(t *testing.T) {
	type lines = []lineRange
	type result struct {
		lang   string
		ranges []lineRange
	}

	tests := map[string]result{
		"js":               {"js", lines{}},
		"js/":              {"js", lines{}},
		"js/0-1":           {"js", lines{}},
		"js/5-4":           {"js", lines{{5, 5}}},
		"js/5 - ":          {"js", lines{{5, 5}}},
		"js/5":             {"js", lines{{5, 5}}},
		"py/1-5":           {"py", lines{{1, 5}}},
		"rs/1,4":           {"rs", lines{{1, 1}, {4, 4}}},
		"tsx/1-2, 5, 9-20": {"tsx", lines{{1, 2}, {5, 5}, {9, 20}}},
		"tsx/1-2-3":        {"tsx", lines{{1, 2}}},
	}

	for input, expected := range tests {
		actualLang, actualRanges := parseHighlightRanges(input)

		if actualLang != expected.lang {
			t.Errorf(`expected language in "%s" to be "%s" but got "%s"`, input, expected.lang, actualLang)
		}

		if !reflect.DeepEqual(expected.ranges, actualRanges) {
			t.Errorf(`expected ranges in "%s" to be "%v" but got "%v"`, input, expected.ranges, actualRanges)
		}
	}
}

var markdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Footnote,
		NewHighlighting(
			// TODO: This should be configurable. How to avoid without config file?
			"algol_nu",
			// TODO: This should be turned off for CSS themes
			//chromaHtml.WithClasses(true),
			chromaHtml.TabWidth(2),
		),
	),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
)

//go:embed highlighting_test.go
var src []byte
var md = fmt.Sprintf("```go\n%s\n```", src)

func BenchmarkSyntaxHighlighting(b *testing.B) {
	for n := 0; n <= b.N; n++ {
		var buf bytes.Buffer
		markdown.Convert([]byte(md), &buf)
	}
}
