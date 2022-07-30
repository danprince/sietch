package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type testFS = map[string]string

func setup(t *testing.T, fileMap testFS) builder {
	tmpDir := path.Join("/tmp/", fmt.Sprintf("%x", rand.Intn(100_000)))

	for filePath, contents := range fileMap {
		absPath := path.Join(tmpDir, filePath)
		dir := path.Dir(absPath)
		err := os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
		err = os.WriteFile(absPath, []byte(contents), 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return builder{
		rootDir:      tmpDir,
		pagesDir:     tmpDir,
		outDir:       path.Join(tmpDir, "_site"),
		templateFile: path.Join(tmpDir, "_template.html"),
		configFile:   path.Join(tmpDir, ".sietch.json"),
	}
}

func (b *builder) findPageByPath(t *testing.T, p string) *Page {
	for _, page := range b.pages {
		if page.path == p {
			return page
		}
	}

	t.Fatalf(`expected page to exist "%s"`, p)
	return nil
}

func TestBuild(t *testing.T) {
	b := setup(t, testFS{
		"index.md": "# Index",
	})

	_, err := b.build()

	if err != nil {
		t.Errorf("expected to build without errors: %s", err)
	}

	if len(b.pages) != 1 {
		t.Errorf("expected to build 1 page, not %d", len(b.pages))
	}

	if len(b.assets) > 0 {
		t.Errorf("expected to copy zero assets, not %d", len(b.assets))
	}

	if _, err := os.Stat(b.outDir); err != nil {
		t.Errorf("expected %s to have been created", b.outDir)
	}

	tabrJsPath := path.Join(b.outDir, "index.html")
	indexHtmlContent, err := os.ReadFile(tabrJsPath)

	if err != nil {
		t.Errorf("expected %s to exist and be readable: %s", tabrJsPath, err)
	}

	search := `<h1 id="index">Index</h1>`
	html := string(indexHtmlContent)
	if !strings.Contains(html, search) {
		t.Errorf(`expected file to contain "%s" but got %s`, search, html)
	}
}

func TestBuildAssets(t *testing.T) {
	b := setup(t, testFS{
		"index.md": "# Index",
		"tabr.js":  "alert('stilgar')",
	})

	_, err := b.build()

	if err != nil {
		t.Errorf("expected to build without errors: %s", err)
	}

	if len(b.assets) != 1 {
		t.Errorf("expected to copy 1 asset, not %d", len(b.assets))
	}

	tabrJsPath := path.Join(b.outDir, "tabr.js")
	tabrJsContent, err := os.ReadFile(tabrJsPath)

	if err != nil {
		t.Errorf(`expected "%s" to exist and be readable: %s`, tabrJsPath, err)
	}

	actualContent := string(tabrJsContent)
	expectedContent := "alert('stilgar')"

	if actualContent != expectedContent {
		t.Errorf(`expected "%s" to contain "%s", but found "%s"`, tabrJsPath, expectedContent, actualContent)
	}
}

func TestIgnorePaths(t *testing.T) {
	b := setup(t, testFS{
		"index.md":        "# Index",
		"_nope.md":        "# Nope",
		"_nested/nope.md": "# Nested Nope",
	})

	_, err := b.build()

	if err != nil {
		t.Errorf("expected to build without errors: %s", err)
	}

	ignoredPaths := []string{
		path.Join(b.outDir, "_nope.html"),
		path.Join(b.outDir, "_nested/_nope.html"),
	}

	for _, p := range ignoredPaths {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("expected %s to be ignored", p)
		}
	}
}

func TestFrontMatter(t *testing.T) {
	b := setup(t, testFS{
		"index.md": `---
str: page
num: 35
text: |
  hello
---

Cool!
`,
	})

	_, err := b.build()

	if err != nil {
		t.Errorf("expected to build without errors: %s", err)
	}

	p := b.pages[0]

	tests := map[string]any{
		"str":  "page",
		"num":  35,
		"text": "hello\n",
	}

	for k, expected := range tests {
		actual := p.Data[k]
		if actual != expected {
			t.Errorf(`expected "%s" to parse as "%v" but got "%v"`, k, expected, actual)
		}
	}
}

func TestLocalTemplates(t *testing.T) {
	b := setup(t, testFS{
		"index.md": `---
id: tabr
---

<pre>{{ .Data.id }}</pre>
`,
	})

	_, err := b.build()

	if err != nil {
		t.Errorf("expected to build without errors: %s", err)
	}

	p := b.pages[0]

	searchHtml := "<pre>tabr</pre>"
	actualHtml := p.Contents

	if !strings.Contains(actualHtml, searchHtml) {
		t.Errorf(`did not find "%s" in html %s%s`, searchHtml, "\n", actualHtml)
	}
}

func TestPageUrls(t *testing.T) {
	b := setup(t, testFS{
		"index.md":      ``,
		"docs/index.md": ``,
		"docs/help.md":  ``,
	})

	_, err := b.build()

	if err != nil {
		t.Errorf("expected to build without errors: %s", err)
	}

	tests := map[string]string{
		"/index.md":      "/",
		"/docs/index.md": "/docs/",
		"/docs/help.md":  "/docs/help.html",
	}

	for p, url := range tests {
		page := b.findPageByPath(t, p)
		if page.Url != url {
			t.Errorf(`page "%s" expected to have url "%s" but instead has "%s"`, p, url, page.Url)
		}
	}
}

func TestConfigFile(t *testing.T) {
	b := setup(t, testFS{
		"index.md":     "",
		".sietch.json": `{"SyntaxColor": "monokai"}`,
	})

	_, err := b.build()

	if err != nil {
		t.Errorf("expected to build without errors: %s", err)
	}

	expectedConfig := config{
		SyntaxColor: "monokai",
	}

	if !reflect.DeepEqual(b.config, expectedConfig) {
		t.Errorf("expected config %+v but got %+v", expectedConfig, b.config)
	}
}

func TestCustomTemplate(t *testing.T) {
	indexMd := `
---
title: Fremmen
---
Secrecy`

	templateHtml := `<h1>{{ .Data.title }}</h1>
<main>{{ .Contents }}</main>`

	b := setup(t, testFS{
		"index.md":       indexMd,
		"_template.html": templateHtml,
	})

	_, err := b.build()

	if err != nil {
		t.Errorf("expected to build without errors: %s", err)
	}

	p := b.findPageByPath(t, "/index.md")

	actualHtml := p.Contents
	expectedHtml := `<h1>Fremmen</h1>
<main><p>Secrecy</p>
</main>`

	if actualHtml != expectedHtml {
		t.Errorf("expected custom template to render:\n%s\nbut got\n%s", expectedHtml, actualHtml)
	}
}

func TestDateFormat(t *testing.T) {
	b := setup(t, testFS{
		"index.md": `
---
date: 2022-10-10
---
<time>{{ .Date.Format "Jan 2, 2006" }}</time>`,
	})

	_, err := b.build()

	if err != nil {
		t.Errorf("expected to build without errors: %s", err)
	}

	p := b.findPageByPath(t, "/index.md")

	actualHtml := p.Contents
	searchHtml := `<time>Oct 10, 2022</time>`

	if !strings.Contains(actualHtml, searchHtml) {
		t.Errorf(`did not find "%s" in html %s%s`, searchHtml, "\n", actualHtml)
	}
}

func TestIndexPage(t *testing.T) {
	b := setup(t, testFS{
		"index.md": `
---
index: true
---`,
		"a.md":       "A",
		"b.md":       "B",
		"c/index.md": "C",
	})

	_, err := b.build()

	if err != nil {
		t.Errorf("expected to build without errors: %s", err)
	}

	p := b.findPageByPath(t, "/index.md")
	html := p.Contents
	expectedHrefs := []string{"/a.html", "/b.html", "/c/"}

	for _, href := range expectedHrefs {
		attr := fmt.Sprintf(`href="%s"`, href)
		if !strings.Contains(html, attr) {
			t.Errorf("expected %s to contain %s\n%s", p.outPath, attr, html)
		}
	}
}

func (b *builder) expectBuildError(t *testing.T, msg string, patterns []string) {
	_, err := b.build()

	if err == nil {
		t.Error(msg)
	}

	message := err.Error()
	failed := false

	for _, pattern := range patterns {
		ok, err := regexp.MatchString(pattern, message)

		if !ok || err != nil {
			t.Errorf("expected error to match /%s/.", pattern)
			failed = true
		}
	}

	if failed {
		t.Log(message)
	}
}

func TestYamlParseError(t *testing.T) {
	b := setup(t, testFS{
		"index.md": `
---
index: |
		<- that is a tab
---`,
	})

	b.expectBuildError(t, "expected yaml parsing to fail", []string{
		"index.md:2",
		"found a tab character where an indentation space is expected",
		"\\^\\^\\^",
	})
}

func TestPageTemplateParseError(t *testing.T) {
	b := setup(t, testFS{
		"index.md": `{{ hello }}`,
	})

	b.expectBuildError(t, "expected template parsing to fail", []string{
		"index.md:1",
		`function "hello" not defined`,
		"\\^\\^\\^",
	})
}

func TestPageTemplateEvaluation(t *testing.T) {
	b := setup(t, testFS{
		"index.md": "{{ .Foo }}",
	})

	b.expectBuildError(t, "expected template evaluation to fail", []string{
		"index.md:1",
		`can't evaluate field Foo`,
		"\\^\\^\\^",
	})
}

func TestTemplateParseError(t *testing.T) {
	b := setup(t, testFS{
		"index.md":       ``,
		"_template.html": "{{hello}}",
	})

	b.expectBuildError(t, "expected template parsing to fail", []string{
		"_template.html:1",
		`function "hello" not defined`,
		"\\^\\^\\^",
	})
}

func TestTemplateEvaluationError(t *testing.T) {
	b := setup(t, testFS{
		"index.md":       ``,
		"_template.html": "{{ .Woo }}",
	})

	b.expectBuildError(t, "expected template evaluation to fail", []string{
		"_template.html:1",
		`can't evaluate field Woo`,
		"\\^\\^\\^",
	})
}

func TestParseConfigError(t *testing.T) {
	b := setup(t, testFS{
		".sietch.json": `{"SyntaxColor":123}`,
	})

	b.expectBuildError(t, "json error", []string{
		`\.sietch\.json`,
		`\^\^\^`,
		"expected config.SyntaxColor to be a string",
	})
}

func TestInvalidConfigError(t *testing.T) {
	b := setup(t, testFS{
		".sietch.json": `{"SyntaxColor":"blebby"}`,
	})

	b.expectBuildError(t, "json error", []string{
		"invalid syntax color: blebby",
	})
}

func TestInvalidSyntaxSuggestions(t *testing.T) {
	b := setup(t, testFS{
		".sietch.json": `{"SyntaxColor":"dragula"}`,
	})

	b.expectBuildError(t, "syntax suggestion error", []string{
		"dracula",
	})
}

func setupBench(b *testing.B, subdir string) builder {
	cwd, _ := os.Getwd()
	tmpDir := path.Join(cwd, subdir)
	return builder{
		rootDir:      tmpDir,
		pagesDir:     tmpDir,
		outDir:       path.Join(tmpDir, "_site"),
		templateFile: path.Join(tmpDir, "_template.html"),
	}
}

func BenchmarkSmall(b *testing.B) {
	builder := setupBench(b, "benchmark/small")

	for n := 0; n < b.N; n++ {
		builder.build()
		builder.reset()
	}
}
func BenchmarkMedium(b *testing.B) {
	builder := setupBench(b, "benchmark/medium")

	for n := 0; n < b.N; n++ {
		builder.build()
		builder.reset()
	}
}

func BenchmarkLarge(b *testing.B) {
	builder := setupBench(b, "benchmark/large")

	for n := 0; n < b.N; n++ {
		builder.build()
		builder.reset()
	}
}

func BenchmarkHuge(b *testing.B) {
	builder := setupBench(b, "benchmark/huge")

	for n := 0; n < b.N; n++ {
		builder.build()
		builder.reset()
	}
}
