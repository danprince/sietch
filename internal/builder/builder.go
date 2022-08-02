package builder

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/danprince/sietch/internal/mdext"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"golang.org/x/sync/errgroup"
	"rogchap.com/v8go"
)

var iso = v8go.NewIsolate()

//go:embed template.html
var defaultTemplateHtml []byte

// Manages the state of the site throughout the duration of the process.
type Builder struct {
	RootDir      string
	PagesDir     string
	AssetsDir    string
	OutDir       string
	template     *template.Template
	templateFile string
	pages        []*Page
	index        map[string][]*Page
	markdown     goldmark.Markdown
	framework    Framework
}

// Frameworks decide how to create the entry point files for bundling islands.
type Framework struct {
	staticEntryPoint func(b *Builder) string
	clientEntryPoint func(b *Builder, p *Page) string
}

type IslandType uint8

const (
	IslandStatic IslandType = iota
	IslandClient
	IslandClientWhenVisible
	IslandClientWhenIdle
	IslandClientOnly
)

type Island struct {
	Id         string
	Marker     string
	Type       IslandType
	Props      map[string]any
	EntryPoint string
}

// Helper for templates that turns an island into HTML.
func (i *Island) String() string {
	switch i.Type {
	case IslandStatic:
		return i.Marker
	case IslandClientOnly:
		return fmt.Sprintf(`<div id="%s"></div>`, i.Id)
	default:
		return fmt.Sprintf(`<div id="%s">%s</div>`, i.Id, i.Marker)
	}
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
	islands          []*Island
}

// Creates a new island and adds it to the page.
func (p *Page) addIsland(entryPoint string, props map[string]any) *Island {
	id := fmt.Sprintf("%s_%d", p.id, len(p.islands))
	marker := fmt.Sprintf("<!-- %s -->", id)

	island := &Island{
		Id:         id,
		Marker:     marker,
		Props:      props,
		EntryPoint: entryPoint,
		Type:       IslandStatic,
	}

	p.islands = append(p.islands, island)
	return island
}

// Creates a new builder with the default settings.
func New(dir string) *Builder {
	return &Builder{
		RootDir:      dir,
		PagesDir:     dir,
		OutDir:       path.Join(dir, "_site"),
		AssetsDir:    path.Join(dir, "_site/_assets"),
		templateFile: path.Join(dir, "_template.html"),
		pages:        []*Page{},
		index:        map[string][]*Page{},
		framework:    Vanilla,
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

	err = b.renderIslands()
	if err != nil {
		return err
	}

	err = b.bundleIslands()
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
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			mdext.ExternalLinks,
			mdext.HeadingAnchors,
			mdext.NewSyntaxHighlighting("algol_nu"),
		),
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
		"props": func(kvs ...any) map[string]any {
			p := make(map[string]any, len(kvs)/2)

			for i := 0; i < len(kvs)-1; i++ {
				k := kvs[i]
				v := kvs[i+1]
				if key, ok := k.(string); ok {
					p[key] = v
				}
			}

			return p
		},
		"render": func(entryPoint string, props map[string]any) *Island {
			// Give all islands absolute entrypoints to keep things simpler later
			if entryPoint[0] == '.' {
				entryPoint = path.Join(b.PagesDir, page.Dir, entryPoint)
			}

			return page.addIsland(entryPoint, props)
		},
		"clientOnly": func(island *Island) *Island {
			island.Type = IslandClientOnly
			return island
		},
		"clientEager": func(island *Island) *Island {
			island.Type = IslandClient
			return island
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
	return filepath.WalkDir(b.PagesDir, func(p string, d fs.DirEntry, err error) error {
		name := d.Name()

		// Skip over ignored files
		if name[0] == '_' || name[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir
			} else {
				return err
			}
		}

		rel := strings.TrimPrefix(p, b.PagesDir)
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
	inputPath := path.Join(b.PagesDir, relPath)
	outputPath := path.Join(b.OutDir, out)

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
	b.pages = append(b.pages, page)
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
			if t, err := time.Parse("2006-1-2", s); err == nil {
				page.Date = t
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

// Render the islands which produce static HTML into their respective pages.
func (b *Builder) renderIslands() error {
	code := b.framework.staticEntryPoint(b)

	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   code,
			Sourcefile: "server-entry.js",
			ResolveDir: b.PagesDir,
		},
		Outdir:    b.AssetsDir,
		Bundle:    true,
		Write:     false,
		Platform:  api.PlatformNeutral,
		Format:    api.FormatIIFE,
		Sourcemap: api.SourceMapExternal,
	})

	if len(result.Errors) > 0 {
		return errors.New(result.Errors[0].Text)
	}

	var source []byte
	var sourceMap []byte

	for _, file := range result.OutputFiles {
		switch path.Base(file.Path) {
		case "stdin.js":
			source = file.Contents
		case "stdin.js.map":
			sourceMap = file.Contents
		}
	}

	script := fmt.Sprintf("globalThis.$elements = {};\n%s\n$elements", string(source))

	ctx := v8go.NewContext(iso)
	val, err := ctx.RunScript(script, "server-entry.js")

	if err != nil {
		fmt.Println(source, sourceMap)
		return err
	}

	s, err := v8go.JSONStringify(ctx, val)

	if err != nil {
		return err
	}

	elements := map[string]string{}
	err = json.Unmarshal([]byte(s), &elements)

	if err != nil {
		return err
	}

	for _, page := range b.pages {
		for _, island := range page.islands {
			if html, ok := elements[island.Id]; ok {
				page.Contents = strings.Replace(page.Contents, island.Marker, html, 1)
			}
		}
	}

	return nil
}

// Create client side bundles for the dynamic islands and inject their scripts
// and styles into pages as necessary.
func (b *Builder) bundleIslands() error {
	entryPoints := []string{}

	for _, page := range b.pages {
		for _, island := range page.islands {
			if island.Type != IslandStatic {
				entryPoints = append(entryPoints, fmt.Sprintf("%s?browser", page.id))
				break
			}
		}
	}

	result := api.Build(api.BuildOptions{
		EntryPoints: entryPoints,
		Bundle:      true,
		Write:       true,
		Outdir:      b.AssetsDir,
		Platform:    api.PlatformBrowser,
		Sourcemap:   api.SourceMapLinked,
		Format:      api.FormatESModule,
		Splitting:   true,
		Plugins: []api.Plugin{
			browserPagesPlugin(b),
		},
	})

	if len(result.Errors) > 0 {
		return errors.New(result.Errors[0].Text)
	}

	// Esbuild turns the ?browser flag we add to our entrypoints into a _browser
	// suffix, so we can work out which files came from which pages by removing
	// this suffix to get the page ID.
	suffix := regexp.MustCompile(`_browser\.(js|css)$`)

	type bundle struct {
		styles  []string
		scripts []string
	}

	bundles := map[string]bundle{}

	for _, file := range result.OutputFiles {
		href := strings.TrimPrefix(file.Path, b.AssetsDir)
		name := path.Base(file.Path)
		pageId := suffix.ReplaceAllString(name, "")

		if _, ok := bundles[pageId]; !ok {
			bundles[pageId] = bundle{}
		}

		if bundle, ok := bundles[pageId]; ok {
			switch path.Ext(name) {
			case ".js":
				bundle.scripts = append(bundle.scripts, href)
			case ".css":
				bundle.styles = append(bundle.styles, href)
			}
		}
	}

	for _, page := range b.pages {
		if bundle, ok := bundles[page.id]; ok {
			var scriptTags strings.Builder
			var linkTags strings.Builder

			for _, src := range bundle.scripts {
				scriptTags.WriteString(fmt.Sprintf(`<script type="module" src="%s"></script>`, src))
				scriptTags.WriteByte('\n')
			}

			for _, href := range bundle.styles {
				linkTags.WriteString(fmt.Sprintf(`<link rel="stylesheet" href="%s">`, href))
				linkTags.WriteByte('\n')
			}

			page.Contents = strings.Replace(page.Contents, "</head>", linkTags.String()+"</head>", 1)
			page.Contents = strings.Replace(page.Contents, "</body>", scriptTags.String()+"</body>", 1)
		}
	}

	return nil
}

// Writes all files in the site into the output directory.
func (b *Builder) writeFiles() error {
	for _, page := range b.pages {
		dir := path.Join(b.OutDir, page.Dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(page.outputPath, []byte(page.Contents), 0755); err != nil {
			return err
		}
	}

	return nil
}
