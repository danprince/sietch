package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/adrg/frontmatter"
	chromaHtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/danprince/sietch/internal/errors"
	"github.com/danprince/sietch/internal/markdown"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"golang.org/x/sync/errgroup"
)

//go:embed template.html
var defaultTemplateHtml []byte

// Builder's hold all the necessary information to produce a static site.
type builder struct {
	// Working directory the command was run.
	rootDir string
	// Directory to start scan for markdown pages.
	pagesDir string
	// Directory to output content (defaults to <rootDir>/_site).
	outDir string
	// Path to the template file (defaults to <rootDir>/.sietch.json).
	configFile string
	// Parsed version of configFile
	config config
	// Path to the template file (defaults to <rootDir>/_template.html).
	templateFile string
	// Compiled version of the site's template file.
	template *template.Template
	// Relative paths to all non-ignored directories in the pagesDir.
	dirs []string
	// All the pages in the site.
	pages []*Page
	// A nested slice of pages by directory depth.
	pagesByDepth [][]*Page
	// All non-page assets in the site.
	assets []*Asset
	// Configured markdown parser/renderer
	markdown goldmark.Markdown
}

// A page represents a single markdown file in the pagesDir.
type Page struct {
	path           string
	Name           string
	dir            string
	Url            string
	Contents       string
	Data           map[string]any
	Date           time.Time
	depth          int
	outPath        string
	contentsOffset int
}

// An asset is a non-markdown file that will be copied directly to the output.
type Asset struct {
	Path string
}

// Reset the internal state of the builder to prevent memory leaks across
// successive rebuilds.
func (b *builder) reset() {
	b.dirs = nil
	b.pages = nil
	b.pagesByDepth = nil
	b.assets = nil
}

// Scan, parse, and compile the entire site.
func (b *builder) build() (time.Duration, error) {
	var err error
	var dt time.Duration
	start := time.Now()

	b.scan()

	err = b.readConfig()
	if err != nil {
		return dt, err
	}

	b.setup()

	err = b.readTemplates()
	if err != nil {
		return dt, err
	}

	err = b.readPages()
	if err != nil {
		return dt, err
	}

	err = b.mkdirs()
	if err != nil {
		return dt, err
	}

	err = b.buildPages()
	if err != nil {
		return dt, err
	}

	err = b.buildAssets()
	if err != nil {
		return dt, err
	}

	dt = time.Since(start)
	return dt, nil
}

func (b *builder) setup() {
	defaultStyle := "doom-one"
	syntaxStyle := b.config.SyntaxColor
	withClasses := false

	if syntaxStyle == "css" {
		syntaxStyle = defaultStyle
		withClasses = true
	}

	b.markdown = goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			markdown.Links,
			markdown.NewHighlighting(
				syntaxStyle,
				chromaHtml.WithClasses(withClasses),
				chromaHtml.TabWidth(2),
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
}

// Recursive walk of the site to identify pages and assets.
func (b *builder) scan() error {
	return filepath.WalkDir(b.pagesDir, func(absPath string, entry fs.DirEntry, err error) error {
		name := entry.Name()

		if name[0] == '_' || name[0] == '.' {
			if entry.IsDir() {
				return filepath.SkipDir
			} else {
				return err
			}
		}

		relPath := strings.TrimPrefix(absPath, b.pagesDir)
		depth := strings.Count(relPath, "/")
		dir := path.Dir(relPath)

		if entry.IsDir() {
			b.dirs = append(b.dirs, relPath)
		} else if strings.HasSuffix(name, ".md") {
			outPath := strings.Replace(relPath, ".md", ".html", 1)
			url := strings.Replace(outPath, "index.html", "", 1)

			// Index files are an edge case because they always need to be built
			// last for a given directory and the should usually show up as a page
			// in the parent dir, rather than in their own dir (listing themselves).
			if name == "index.md" {
				depth -= 1
			}

			for len(b.pagesByDepth) <= depth {
				b.pagesByDepth = append(b.pagesByDepth, []*Page{})
			}

			page := Page{
				Url:     url,
				dir:     dir,
				Name:    name,
				path:    relPath,
				depth:   depth,
				outPath: outPath,
			}

			b.pages = append(b.pages, &page)
			b.pagesByDepth[depth] = append(b.pagesByDepth[depth], &page)
		} else {
			b.assets = append(b.assets, &Asset{Path: relPath})
		}

		return err
	})
}

// Creates the default set of functions that will be available in any page
// templates.
func (b *builder) defaultTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"include": func(name string) string {
			contents, err := os.ReadFile(name)
			if err != nil {
				return err.Error()
			} else {
				return string(contents)
			}
		},
		"date": func(layout, format, date string) string {
			t, err := time.Parse(layout, date)
			if err != nil {
				fmt.Fprintln(os.Stderr, errors.FmtError(err))
				return "Invalid date"
			}
			return t.Format(format)
		},
		"home": func(page *Page) *Page {
			if page.Name == "index.md" {
				return page
			}
			for _, p := range b.pagesByDepth[page.depth-1] {
				if p.Name == "index.md" {
					return p
				}
			}
			return nil
		},
		"index": func(page *Page) []*Page {
			var siblings []*Page
			depth := page.depth

			if page.Name == "index.md" {
				depth += 1
			}

			for _, other := range b.pagesByDepth[depth] {
				if other.dir == page.dir || path.Dir(other.dir) == page.dir {
					siblings = append(siblings, other)
				}
			}

			sort.SliceStable(siblings, func(i, j int) bool {
				return siblings[i].Date.Before(siblings[j].Date)
			})

			return siblings
		},
		"attrs": func(kvs ...any) map[string]any {
			m := make(map[string]any, len(kvs)/2)

			for i, k := range kvs {
				val := kvs[i+1]
				key, ok := k.(string)
				if ok {
					m[key] = val
				}
			}

			return m
		},
	}
}

// Reads the config file (if it exists)
func (b *builder) readConfig() error {
	_, err := os.Stat(b.configFile)

	if err != nil {
		return nil
	}

	contents, err := os.ReadFile(b.configFile)

	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	err = json.Unmarshal(contents, &b.config)
	file := strings.TrimPrefix(b.configFile, b.pagesDir)

	err = errors.ParseJsonError(err, file, string(contents))
	if err != nil {
		return err
	}

	return b.config.validate()
}

// Reads the appropriate page templates from disk, and falls back to defaults
// if they don't exist.
func (b *builder) readTemplates() error {
	contents, err := os.ReadFile(b.templateFile)

	if err != nil {
		contents = defaultTemplateHtml
		b.templateFile = "template.html"
	}

	funcs := b.defaultTemplateFuncs()
	template, err := template.New("template").Funcs(funcs).Parse(string(contents))

	if err != nil {
		return errors.TemplateParseError(err, b.templateFile, string(contents), 0)
	}

	b.template = template

	return nil
}

// Reads all pages concurrently.
func (b *builder) readPages() error {
	var g errgroup.Group
	for _, p := range b.pages {
		page := p
		g.Go(func() error {
			return b.readPage(page)
		})
	}
	return g.Wait()
}

var builtinDateFormats = []string{
	"2006-1-2",
}

// Reads the contents and parses the front matter for a page.
func (b *builder) readPage(page *Page) error {
	name := path.Join(b.pagesDir, page.path)
	contents, err := os.ReadFile(name)

	if err != nil {
		return err
	}

	r := bytes.NewReader(contents)
	markdown, err := frontmatter.Parse(r, &page.Data)

	if err != nil {
		return errors.YamlParseError(err, page.path, string(contents))
	}

	// Attempt to parse a real date from the front matter
	anyDate := page.Data["date"]

	if anyDate != nil {
		dateStr, ok := anyDate.(string)
		if ok {
			for _, layout := range builtinDateFormats {
				date, err := time.Parse(layout, dateStr)
				if err == nil {
					page.Date = date
					break
				}
			}
		}
	}

	// Calculate how much the frontmatter offset the page contents to help with
	// showing meaningful errors later.
	frontMatterEndOffset := len(contents) - len(markdown)
	for i := 0; i < frontMatterEndOffset; i++ {
		if contents[i] == '\n' {
			page.contentsOffset += 1
		}
	}

	page.Contents = string(markdown)
	return nil
}

// Recreates all scanned directories in the output dir, so that files can be
// written there without worrying about non-existent paths.
func (b *builder) mkdirs() error {
	for _, dir := range b.dirs {
		err := os.MkdirAll(path.Join(b.outDir, dir), os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

// Build and render all pages concurrently
func (b *builder) buildPages() error {
	var g errgroup.Group

	for _, p := range b.pages {
		page := p
		g.Go(func() error {
			return b.buildPage(page)
		})
	}

	return g.Wait()
}

// Converts markdown to html and renders the contents inside the site's page
// template.
func (b *builder) buildPage(page *Page) error {
	// Parse and execute the page's own template
	text := page.Contents
	funcs := b.defaultTemplateFuncs()
	tpl, err := template.New("page").Funcs(funcs).Parse(text)

	if err != nil {
		return errors.TemplateParseError(err, page.path, page.Contents, page.contentsOffset)
	}

	// Context used for executing the page and theme templates
	ctx := page

	var mdBuf bytes.Buffer
	err = tpl.Execute(&mdBuf, ctx)

	if err != nil {
		return errors.TemplateExecError(err, page.path, page.Contents, page.contentsOffset)
	}

	// Convert the result of the template from markdown to html
	var htmlBuf bytes.Buffer
	err = b.markdown.Convert(mdBuf.Bytes(), &htmlBuf)

	if err != nil {
		return err
	}

	page.Contents = htmlBuf.String()

	// Pass into the main template
	var pageBuf bytes.Buffer
	err = b.template.Execute(&pageBuf, ctx)

	if err != nil {
		return errors.TemplateExecError(err, b.templateFile, string(defaultTemplateHtml), 0)
	}

	page.Contents = pageBuf.String()

	// Finally, write to disk
	err = os.WriteFile(path.Join(b.outDir, page.outPath), pageBuf.Bytes(), os.ModePerm)

	if err != nil {
		return err
	}

	return nil
}

// Copies all non-ignored assets into the output directory concurrently.
func (b *builder) buildAssets() error {
	var g errgroup.Group
	for _, a := range b.assets {
		asset := a
		g.Go(func() error {
			return b.buildAsset(asset)
		})
	}
	return g.Wait()
}

// Copies the asset into the output directory.
func (b *builder) buildAsset(asset *Asset) error {
	src := path.Join(b.pagesDir, asset.Path)
	dst := path.Join(b.outDir, asset.Path)

	srcFile, err := os.Open(src)

	if err != nil {
		return err
	}

	defer srcFile.Close()

	dstFile, err := os.Create(dst)

	if err != nil {
		return err
	}

	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)

	if err != nil {
		return err
	}

	return nil
}
