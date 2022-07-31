package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestHeadings(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(Headings))

	tests := map[string]string{
		`# H1`:                     `<a href="#h1" class="permalink"><h1 id="h1">H1</h1></a>`,
		`## H2`:                    `<a href="#h2" class="permalink"><h2 id="h2">H2</h2></a>`,
		`### H3`:                   `<a href="#h3" class="permalink"><h3 id="h3">H3</h3></a>`,
		`#### H4`:                  `<a href="#h4" class="permalink"><h4 id="h4">H4</h4></a>`,
		`##### H5`:                 `<a href="#h5" class="permalink"><h5 id="h5">H5</h5></a>`,
		`###### H6`:                `<a href="#h6" class="permalink"><h6 id="h6">H6</h6></a>`,
		`# Many Words`:             `<a href="#many-words" class="permalink"><h1 id="many-words">Many Words</h1></a>`,
		"# Collision\n# Collision": `<a href="#collision" class="permalink"><h1 id="collision">Collision</h1></a><a href="#collision-1" class="permalink"><h1 id="collision-1">Collision</h1></a>`,
	}

	for input, expected := range tests {
		var buf bytes.Buffer
		err := md.Convert([]byte(input), &buf)
		actual := strings.TrimSpace(buf.String())

		if err != nil {
			t.Errorf("unexpected markdown error: %s", err)
		}

		if actual != expected {
			t.Errorf("expected \n\"%s\",\n\"%s\"", expected, actual)
		}
	}
}
