package builder

import (
	"bytes"
	_ "embed"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
	"golang.org/x/sync/errgroup"
)

//go:embed template.html
var defaultTemplateHtml []byte

// Manages the state of the site throughout the duration of the process.
type Builder struct {
	rootDir      string
	pagesDir     string
	outDir       string
	template     *template.Template
	templateFile string
	pages        map[string]*Page
	index        map[string][]*Page
	markdown     goldmark.Markdown
}

// Page is a markdown file in the site.
type Page struct {
	id               string
	Path             string
	Dir              string
	Url              string
	Data             map[string]any
	Date             time.Time
	Contents         string
	template         *template.Template
	inputPath        string
	outputPath       string
	contentStartLine int
}

// Creates a new builder with the default settings.
func New(dir string) *Builder {
	return &Builder{
		rootDir:      dir,
		pagesDir:     dir,
		outDir:       path.Join(dir, "_site"),
		templateFile: path.Join(dir, "_template.html"),
		pages:        map[string]*Page{},
		index:        map[string][]*Page{},
	}
}

// Resets the state of a builder to prevent leaking memory across builds.
func (b *Builder) Reset() {
	b.pages = nil
	b.index = nil
}

// Builds the site.
func (b *Builder) Build() error {
	var err error

	b.configure()

	err = b.readTemplate()
	if err != nil {
		return err
	}

	err = b.walk()
	if err != nil {
		return err
	}

	err = b.readPages()
	if err != nil {
		return err
	}

	err = b.buildPages()
	if err != nil {
		return err
	}

	err = b.writeFiles()
	if err != nil {
		return err
	}

	return nil
}

// Configure everything required to start building.
func (b *Builder) configure() {
	b.markdown = goldmark.New(
		goldmark.WithExtensions(),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
}

// Creates the set of functions used to render page contents.
func (b *Builder) templateFuncs(page *Page) template.FuncMap {
	return template.FuncMap{
		"index": func() []*Page {
			return b.index[page.Dir]
		},
		"orderByDate": func(pages []*Page) []*Page {
			sort.SliceStable(pages, func(i, j int) bool {
				return pages[i].Date.Before(pages[j].Date)
			})
			return pages
		},
		"pagesWith": func(key string) []*Page {
			var pages []*Page
			for _, page := range b.pages {
				if page.Data[key] != nil {
					pages = append(pages, page)
				}
			}
			return pages
		},
		"sortBy": func(key string, pages []*Page) []*Page {
			sort.SliceStable(pages, func(i, j int) bool {
				a := pages[i].Data[key]
				b := pages[j].Data[key]
				return lessAny(a, b)
			})
			return pages
		},
	}
}

// Read and parse the site's global page template.
func (b *Builder) readTemplate() error {
	contents, err := os.ReadFile(b.templateFile)
	funcs := b.templateFuncs(nil)

	if os.IsNotExist(err) {
		contents = defaultTemplateHtml
	} else if err != nil {
		return err
	}

	t, err := template.New("template").Funcs(funcs).Parse(string(contents))

	if err != nil {
		return err
	}

	b.template = t
	return nil
}

// Recursive walk through the site's pages dir, searching for markdown files
// and adding them to the builder.
func (b *Builder) walk() error {
	return filepath.WalkDir(b.pagesDir, func(p string, d fs.DirEntry, err error) error {
		name := d.Name()

		// Skip over ignored files
		if name[0] == '_' || name[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir
			} else {
				return err
			}
		}

		rel := strings.TrimPrefix(p, b.pagesDir)
		ext := path.Ext(rel)

		if ext == ".md" {
			b.addPage(rel)
		}

		return err
	})
}

// Adds a page to the builder given a path that is relative to the pagesDir.
func (b *Builder) addPage(relPath string) {
	id := shortHash(relPath)
	name := path.Base(relPath)
	dir := path.Dir(relPath)
	out := strings.Replace(relPath, ".md", ".html", 1)
	url := strings.Replace(out, "index.html", "", 1)
	inputPath := path.Join(b.pagesDir, relPath)
	outputPath := path.Join(b.outDir, out)

	page := &Page{
		id:         id,
		Path:       relPath,
		Url:        url,
		Dir:        dir,
		Data:       map[string]any{},
		inputPath:  inputPath,
		outputPath: outputPath,
	}

	parent := dir

	// index.md files are indexed as though they were in the parent directory.
	// (e.g. /posts/hello-world/index.md would be indexed in /posts).
	if name == "index.md" {
		parent = path.Dir(parent)
	}

	if b.index[parent] == nil {
		b.index[parent] = []*Page{}
	}

	b.index[parent] = append(b.index[parent], page)
	b.pages[relPath] = page
}

// Read all pages in the site concurrently.
func (b *Builder) readPages() error {
	var g errgroup.Group
	for _, page := range b.pages {
		p := page
		g.Go(func() error {
			return b.readPage(p)
		})
	}
	return g.Wait()
}

// Read the page's contents and metadata from disk.
func (b *Builder) readPage(page *Page) error {
	rawContents, err := os.ReadFile(page.inputPath)

	if err != nil {
		return err
	}

	r := bytes.NewReader(rawContents)
	contents, err := frontmatter.Parse(r, &page.Data)

	if err != nil {
		return err
	}

	// Figure out number of lines of front matter for line numbers in errors
	frontMatterLen := len(rawContents) - len(contents)
	frontMatterBytes := contents[:frontMatterLen]
	page.contentStartLine = bytes.Count(frontMatterBytes, []byte{'\n'})

	// Contents is everything after the front matter
	page.Contents = string(contents)

	// Parse the page template
	funcs := b.templateFuncs(page)
	tmpl, err := template.New(page.Path).Funcs(funcs).Parse(page.Contents)
	if err != nil {
		return err
	}
	page.template = tmpl

	// Attempt to parse the date from front matter
	date := page.Data["date"]
	if date != nil {
		if s, ok := date.(string); ok {
			if t, err := time.ParseInLocation("2006-1-2", s, time.Local); err == nil {
				page.Date = t.Local()
			}
		}
	}

	return nil
}

// Builds all pages concurrently.
func (b *Builder) buildPages() error {
	var g errgroup.Group
	for _, page := range b.pages {
		p := page
		g.Go(func() error {
			return b.buildPage(p)
		})
	}
	return g.Wait()
}

// Converts the page's markdown into HTML and renders it into the builder's
// template file.
func (b *Builder) buildPage(page *Page) error {
	funcs := b.templateFuncs(page)
	globalTemplate, err := b.template.Funcs(funcs).Parse("")

	if err != nil {
		return err
	}

	var mdbuf, htmlbuf, pagebuf bytes.Buffer

	if err := page.template.Execute(&mdbuf, page); err != nil {
		return err
	}

	if err := b.markdown.Convert(mdbuf.Bytes(), &htmlbuf); err != nil {
		return err
	}

	page.Contents = htmlbuf.String()

	if err := globalTemplate.Execute(&pagebuf, page); err != nil {
		return err
	}

	page.Contents = pagebuf.String()
	return nil
}

// Writes all files in the site into the output directory.
func (b *Builder) writeFiles() error {
	for _, page := range b.pages {
		dir := path.Join(b.outDir, page.Dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(page.outputPath, []byte(page.Contents), 0755); err != nil {
			return err
		}
	}

	return nil
}
