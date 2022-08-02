package builder

import (
	"log"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

// Temporary file system mappings from relative file names to contents
type tfs map[string]string

// Create a builder with a set of initial files in a temporary directory and
// clean it all up when the test stops.
func newTestBuilder(t *testing.T, fs tfs) *Builder {
	root := path.Join(os.TempDir(), t.Name())

	for file, contents := range fs {
		abs := path.Join(root, file)
		dir := path.Dir(abs)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(contents), 0755); err != nil {
			log.Fatal(err)
		}
	}

	t.Cleanup(func() {
		os.RemoveAll(root)
	})

	return New(root)
}

// Expect to find a given string inside the contents of a file, relative
// to the builder's outDir.
func expectInFile(t *testing.T, b *Builder, filename, search string) {
	filename = path.Join(b.OutDir, filename)
	contents, err := os.ReadFile(filename)

	if err != nil {
		t.Fatalf("error reading %s: %s", filename, err)
	}

	body := string(contents)

	if !strings.Contains(body, search) {
		t.Errorf("%s did not contain \"%s\"\n\n%s", filename, search, body)
	}
}

// Expect a page to exist and return it. This throws much friendlier errors
// than de-referencing a nil pointer instead.
func expectPage(t *testing.T, b *Builder, pth string) *Page {
	var page *Page

	for _, p := range b.pages {
		if p.Path == pth {
			page = p
		}
	}

	if page == nil {
		t.Fatalf(`expected page to exist: %s`, pth)
	}
	return page
}

// Run a build and fail the current test if it throws an error.
func buildWithoutErrors(t *testing.T, b *Builder) {
	err := b.Build()

	if err != nil {
		t.Fatalf("build error: %s", err)
	}
}

func TestBuild(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"index.md": "_Hello!_",
	})

	buildWithoutErrors(t, b)

	expectInFile(t, b, "index.html", `<em>Hello!</em>`)

	if len(b.pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(b.pages))
	}
}

func TestIgnorePaths(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"hello.md": "# Hello",
		"_nope.md": "# Nope",
	})

	buildWithoutErrors(t, b)

	if len(b.pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(b.pages))
	}
}

func TestFrontMatter(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"hello.md": `---
str: page
num: 35
text: |
  hello
---`,
	})

	buildWithoutErrors(t, b)

	expected := map[string]any{
		"str":  "page",
		"num":  35,
		"text": "hello\n",
	}

	page := expectPage(t, b, "/hello.md")

	for k, v := range expected {
		if page.Data[k] != v {
			t.Errorf("expected %s to be %v, got %v", k, v, page.Data[k])
		}
	}
}

func TestPageUrls(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"index.md":        "",
		"tabr/index.md":   ``,
		"tabr/stilgar.md": ``,
	})

	buildWithoutErrors(t, b)

	tests := map[string]string{
		"/index.md":        "/",
		"/tabr/index.md":   "/tabr/",
		"/tabr/stilgar.md": "/tabr/stilgar.html",
	}

	for name, url := range tests {
		page := expectPage(t, b, name)
		if page.Url != url {
			t.Errorf(`expected %s, got %s`, url, page.Url)
		}
	}
}

func TestLocalTemplates(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"paul.md": `---
name: Kwisatz Haderach
---
I am the {{.Data.name}}`,
	})

	buildWithoutErrors(t, b)
	expectInFile(t, b, "/paul.html", "I am the Kwisatz Haderach")
}

func TestCustomTemplate(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"_template.html": `
<h1>{{ .Data.title }}</h1>
<main>{{ .Contents }}</main>
`,
		"index.md": `---
title: Dune
---
Nested content`,
	})

	buildWithoutErrors(t, b)
	expectInFile(t, b, "/index.html", "<h1>Dune</h1>")
	expectInFile(t, b, "/index.html", "<main><p>Nested content</p>\n</main>")
}

func TestDateParsing(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"index.md": `---
date: 2022-02-20
---`,
	})

	buildWithoutErrors(t, b)
	page := expectPage(t, b, "/index.md")
	expected := time.Date(2022, 2, 20, 0, 0, 0, 0, time.UTC)
	if !page.Date.Equal(expected) {
		t.Errorf(`expected %s, got %s`, expected, page.Date)
	}
}

func TestIndexTemplateFunc(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"_template.html": `{{ .Contents }}`,
		"a.md": `---
title: A
---`,
		"b.md": `---
title: B
---`,
		"c/index.md": `---
title: C
---`,
		"index.md": `
{{- range index -}}
  {{ .Data.title }}
{{- end -}}`,
	})

	buildWithoutErrors(t, b)
	expectInFile(t, b, "index.html", `ABC`)
}

func TestOrderByDateTemplateFunc(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"_template.html": `{{ .Contents }}`,
		"a.md": `---
title: A
date: 2022-6-6
---`,
		"b.md": `---
title: B
date: 2022-5-5
---`,
		"c/index.md": `---
title: C
date: 2022-4-4
---`,
		"index.md": `
{{- range index | orderByDate -}}
  {{ .Data.title }}
{{- end -}}`,
	})

	buildWithoutErrors(t, b)
	expectInFile(t, b, "index.html", `CBA`)
}

func TestPagesWithTemplateFunc(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"_template.html": `{{ .Contents }}`,
		"a.md": `---
title: A
nav: true
---`,
		"b.md": `---
title: B
nav: 2
---`,
		"c/index.md": `---
title: C
---`,
		"index.md": `
<nav>
{{- range pagesWith "nav" -}}
  {{ .Data.title }}
{{- end -}}
</nav>`,
	})

	buildWithoutErrors(t, b)
	expectInFile(t, b, "index.html", `<nav>AB</nav>`)
}

func TestSortByTemplateFunc(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"_template.html": `{{ .Contents }}`,
		"a.md": `---
title: A
nav: 1
---`,
		"b.md": `---
title: B
nav: 2
---`,
		"c/index.md": `---
title: C
nav: 3
---`,
		"index.md": `
<nav>
{{- range pagesWith "nav" | sortBy "nav" -}}
  {{ .Data.title }}
{{- end -}}
</nav>`,
	})

	buildWithoutErrors(t, b)
	expectInFile(t, b, "index.html", `<nav>ABC</nav>`)
}

func TestStaticIsland(t *testing.T) {
	b := newTestBuilder(t, tfs{
		"hello.ts": `
export let render = ({ name }) => "<h1>" + name + "</h1>";
`,
		"index.md": `{{ render "./hello" (props "name" "dan") }}`,
	})

	buildWithoutErrors(t, b)
	expectInFile(t, b, "index.html", `<h1>dan</h1>`)
}
