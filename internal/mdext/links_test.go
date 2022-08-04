package mdext

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestExternalLinks(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(Links))

	tests := map[string]string{
		// Internal links
		`[relative](./rel.html)`: `<a href="./rel.html">relative</a>`,
		`[absolute](/abs.html)`:  `<a href="/abs.html">absolute</a>`,

		// External links
		`[ext](http://ext.com)`:  `<a href="http://ext.com" target="_blank" rel="noopener noreferrer">ext</a>`,
		`[ext](https://ext.com)`: `<a href="https://ext.com" target="_blank" rel="noopener noreferrer">ext</a>`,
		`[ext](://ext.com)`:      `<a href="://ext.com" target="_blank" rel="noopener noreferrer">ext</a>`,
	}

	for input, expected := range tests {
		var buf bytes.Buffer
		err := md.Convert([]byte(input), &buf)
		actual := strings.TrimSpace(buf.String())
		expected = fmt.Sprintf(`<p>%s</p>`, expected)

		if err != nil {
			t.Errorf("unexpected markdown error: %s", err)
		}

		if actual != expected {
			t.Errorf(`expected "%s", got "%s"`, expected, actual)
		}
	}
}
