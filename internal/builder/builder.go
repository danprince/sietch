package builder

import (
	"bytes"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/danprince/sietch/internal/errors"
	"github.com/danprince/sietch/internal/islands"
	"github.com/danprince/sietch/internal/livereload"
	"github.com/danprince/sietch/internal/mdext"
	"github.com/tdewolff/minify/v2"
	mincss "github.com/tdewolff/minify/v2/css"
	minhtml "github.com/tdewolff/minify/v2/html"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"golang.org/x/sync/errgroup"
)

type Mode uint8

const (
	Development Mode = iota
	Production
)

var (
	//go:embed template.html
	defaultTemplateHtml []byte

	//go:embed template.css
	defaultTemplateCss string

	frameworkMap = map[string]*islands.Framework{
		islands.Vanilla.Id: islands.Vanilla,
		islands.Preact.Id:  islands.Preact,
	}
)

// Manages the state of the site throughout the duration of the process.
type Builder struct {
	RootDir      string
	PagesDir     string
	AssetsDir    string
	OutDir       string
	PublicDir    string
	Mode         Mode
	template     *template.Template
	templateFile string
	config       Config
	configFile   string
	pages        []*Page
	assets       map[string]string
	assetsMu     sync.Mutex
	index        map[string][]*Page
	markdown     goldmark.Markdown
	frameworks   []*islands.Framework
	minifier     *minify.M
	fingerprint  bool
	minify       bool
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
	islands          []*islands.Island
}

// Creates a new island and adds it to the page.
func (p *Page) addIsland(entryPoint string, props islands.Props) *islands.Island {
	id := fmt.Sprintf("%s_%d", p.id, len(p.islands))

	island := &islands.Island{
		Id:         id,
		Props:      props,
		EntryPoint: entryPoint,
		Type:       islands.Static,
	}

	p.islands = append(p.islands, island)
	return island
}

// Creates a new builder with the default settings.
func New(dir string, mode Mode) *Builder {
	min := minify.New()
	min.AddFunc("text/html", minhtml.Minify)
	min.AddFunc("text/css", mincss.Minify)

	return &Builder{
		Mode:         mode,
		RootDir:      dir,
		PagesDir:     dir,
		PublicDir:    path.Join(dir, "public"),
		OutDir:       path.Join(dir, "_site"),
		AssetsDir:    path.Join(dir, "_site/_assets"),
		templateFile: path.Join(dir, "_template.html"),
		configFile:   path.Join(dir, ".sietch.json"),
		config:       defaultConfig,
		pages:        []*Page{},
		index:        map[string][]*Page{},
		assets:       map[string]string{},
		assetsMu:     sync.Mutex{},
		frameworks:   []*islands.Framework{islands.Preact, islands.Vanilla},
		minifier:     min,
		minify:       mode == Production,
		fingerprint:  mode == Production,
	}
}

// Resets the state of a builder to prevent leaking memory across builds.
func (b *Builder) Reset() {
	b.config = defaultConfig
	b.pages = []*Page{}
	b.index = map[string][]*Page{}
	b.assets = map[string]string{}
}

// Builds the site.
func (b *Builder) Build() error {
	var err error

	err = b.readConfig()
	if err != nil {
		return err
	}

	err = b.applyConfig()
	if err != nil {
		return err
	}

	err = b.readTemplate()
	if err != nil {
		return err
	}

	err = b.findAssets()
	if err != nil {
		return err
	}

	err = b.findPages()
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

	if b.Mode == Development {
		b.injectDevScripts()
	}

	if b.minify {
		err := b.minifyPages()
		if err != nil {
			return err
		}
	}

	err = b.writeFiles()
	if err != nil {
		return err
	}

	return nil
}

// Read and parse the site's config file
func (b *Builder) readConfig() error {
	return b.config.load(b.configFile)
}

// Configure everything required to start building.
func (b *Builder) applyConfig() error {
	b.PagesDir = path.Join(b.RootDir, b.config.PagesDir)

	// Should this be done as part of reading the config instead?
	if info, err := os.Stat(b.PagesDir); err != nil || !info.IsDir() {
		return errors.Wrap("config", err)
	}

	b.markdown = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			mdext.Links,
			mdext.HeadingAnchors,
			mdext.NewSyntaxHighlighting(b.config.SyntaxColor),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	return nil
}

func (b *Builder) addAsset(file string) string {
	b.assetsMu.Lock()
	defer b.assetsMu.Unlock()

	if url, ok := b.assets[file]; ok {
		return url
	}

	url, _ := filepath.Rel(b.PagesDir, file)
	url = path.Join("/", url)
	b.assets[file] = url
	return url
}

// Creates the set of functions used to render page contents.
func (b *Builder) templateFuncs(page *Page) template.FuncMap {
	return template.FuncMap{
		"url": func(src string) string {
			file := path.Join(b.PagesDir, page.Dir, src)
			return b.addAsset(file)
		},
		"embed": func(src string) string {
			file := path.Join(path.Dir(page.inputPath), src)
			contents, err := os.ReadFile(file)
			if err != nil {
				panic(err)
			}
			return strings.TrimSpace(string(contents))
		},
		"page": func(src string) *Page {
			file := regexp.MustCompile(`/$`).ReplaceAllString(src, "/index.md")
			file = path.Join(path.Dir(page.inputPath), file)

			for _, p := range b.pages {
				if p.inputPath == file {
					return p
				}
			}

			return nil
		},
		"index": func() []*Page {
			return b.index[page.Dir]
		},
		"orderByDate": func(order string, pages []*Page) []*Page {
			if order != "asc" && order != "desc" {
				panic(fmt.Sprintf(`order must be "asc" or "desc", got "%s"`, order))
			}
			asc := order == "asc"
			sort.SliceStable(pages, func(i, j int) bool {
				if asc {
					return pages[i].Date.Before(pages[j].Date)
				} else {
					return pages[i].Date.After(pages[j].Date)
				}
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
		"props": func(kvs ...any) islands.Props {
			if len(kvs)%2 != 0 {
				panic("unbalanced number of keys/values")
			}

			p := make(islands.Props, len(kvs)/2)

			for i := 0; i < len(kvs)-1; i++ {
				k := kvs[i]
				v := kvs[i+1]
				if key, ok := k.(string); ok {
					p[key] = v
				} else {
					panic(fmt.Sprintf("key is not a string: %v", k))
				}
			}

			return p
		},
		"component": func(entryPoint string, allProps ...islands.Props) *islands.Island {
			if entryPoint[0] == '.' {
				// Make the import relative to the pagesDir, so we can use a consistent
				// resolveDir for all islands when we create the static bundle.
				entryPoint = "." + path.Join(page.Dir, entryPoint)
			}

			var props islands.Props

			if len(allProps) > 0 {
				props = allProps[0]
			} else {
				props = islands.Props{}
			}

			return page.addIsland(entryPoint, props)
		},
		"hydrate": func(island *islands.Island) *islands.Island {
			island.Type = islands.HydrateOnLoad
			return island
		},
		"hydrateOnVisible": func(island *islands.Island) *islands.Island {
			island.Type = islands.HydrateOnVisible
			return island
		},
		"hydrateOnIdle": func(island *islands.Island) *islands.Island {
			island.Type = islands.HydrateOnIdle
			return island
		},
		"clientOnly": func(island *islands.Island) *islands.Island {
			island.ClientOnly = true
			return island
		},
		"defaultStyles": func() string {
			return defaultTemplateCss
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
		return errors.Wrap("template", err)
	}

	t, err := template.New("template").Funcs(funcs).Parse(string(contents))

	if err != nil {
		return errors.TemplateParseError(err, b.templateFile, string(contents), 0)
	}

	b.template = t
	return nil
}

// Patterns to ignore whilst we're searching for pages.
var ignorePatterns = []string{
	`^\.`,
	`^_`,
	`^node_modules$`,
}

// Recursive walk through the site's pages dir, searching for markdown files
// and adding them to the builder.
func (b *Builder) findPages() error {
	err := filepath.WalkDir(b.PagesDir, func(p string, d fs.DirEntry, err error) error {
		name := d.Name()

		for _, re := range ignorePatterns {
			if ok, _ := regexp.MatchString(re, name); ok {
				if d.IsDir() {
					return filepath.SkipDir
				} else {
					return err
				}
			}
		}

		rel := strings.TrimPrefix(p, b.PagesDir)
		ext := path.Ext(rel)

		if ext == ".md" {
			b.addPage(rel)
		}

		return err
	})

	if err != nil {
		return err
	}

	return err
}

// Recursively walks the publicDir (if it exists) searching for files and
// adding them to the builder's asset map.
func (b *Builder) findAssets() error {
	info, err := os.Stat(b.PublicDir)

	// If public dir doesn't exist, or is not a directory, there's nothing to search
	if os.IsNotExist(err) || !info.IsDir() {
		return nil
	} else if err != nil {
		return err
	}

	return filepath.WalkDir(b.PublicDir, func(p string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			file, _ := filepath.Rel(b.PublicDir, p)

			// We can write to assets without the mutex safely here, because we're not using
			// goroutines for walk (at the moment).
			b.assets[p] = file
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

	// Keep paths clean by building each page into its own dir with as index.html
	if !strings.HasSuffix(out, "index.html") {
		out = strings.Replace(out, ".html", "/index.html", 1)
	}

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

	// The root index.md doesn't ever get indexed
	if name != "index.md" || dir != "/" {
		b.index[parent] = append(b.index[parent], page)
	}

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
		return errors.Wrap("builder", err)
	}

	r := bytes.NewReader(rawContents)
	contents, err := frontmatter.Parse(r, &page.Data)

	if err != nil {
		return errors.YamlParseError(err, page.inputPath, string(rawContents))
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
		return errors.TemplateParseError(err, page.inputPath, string(contents), page.contentStartLine)
	}
	page.template = tmpl

	// Attempt to parse the date from front matter
	date := page.Data["date"]
	if date != nil {
		if s, ok := date.(string); ok {
			if t, err := time.Parse(b.config.DateFormat, s); err == nil {
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
	globalTemplate, err := b.template.Clone()
	globalTemplate.Funcs(funcs).Parse("")

	if err != nil {
		return errors.TemplateParseError(err, page.inputPath, page.Contents, page.contentStartLine)
	}

	var mdbuf, htmlbuf, pagebuf bytes.Buffer

	if err := page.template.Execute(&mdbuf, page); err != nil {
		return errors.TemplateExecError(err, page.inputPath, page.Contents, page.contentStartLine)
	}

	if err := b.markdown.Convert(mdbuf.Bytes(), &htmlbuf); err != nil {
		return errors.Wrap("markdown", err)
	}

	page.Contents = htmlbuf.String()

	if err := globalTemplate.Execute(&pagebuf, page); err != nil {
		return errors.TemplateExecError(err, b.templateFile, "", 0)
	}

	page.Contents = pagebuf.String()
	return nil
}

func (b *Builder) staticIslands() []*islands.Island {
	staticIslands := []*islands.Island{}

	for _, p := range b.pages {
		for _, i := range p.islands {
			if !i.ClientOnly {
				staticIslands = append(staticIslands, i)
			}
		}
	}

	return staticIslands
}

func (p *Page) clientIslands() []*islands.Island {
	clientIslands := []*islands.Island{}

	for _, i := range p.islands {
		if i.Type != islands.Static {
			clientIslands = append(clientIslands, i)
		}
	}

	return clientIslands
}

func (b *Builder) renderIslands() error {
	elements, err := islands.Render(islands.RenderOptions{
		Islands:    b.staticIslands(),
		AssetsDir:  b.AssetsDir,
		ResolveDir: b.PagesDir,
		Frameworks: b.frameworks,
		Npm:        b.config.Npm,
		ImportMap:  b.config.ImportMap,
	})

	if err != nil {
		return err
	}

	for _, page := range b.pages {
		for _, island := range page.islands {
			if html, ok := elements[island.Id]; ok {
				page.Contents = strings.Replace(page.Contents, island.Marker(), html, 1)
			}
		}
	}

	return nil
}

func (b *Builder) bundleIslands() error {
	islandsByPage := map[string][]*islands.Island{}

	for _, p := range b.pages {
		clientIslands := p.clientIslands()
		if len(clientIslands) > 0 {
			islandsByPage[p.id] = p.clientIslands()
		}
	}

	bundles, err := islands.Bundle(islands.BundleOptions{
		Frameworks:    b.frameworks,
		IslandsByPage: islandsByPage,
		Production:    b.Mode == Production,
		OutDir:        b.OutDir,
		AssetsDir:     b.AssetsDir,
		ResolveDir:    b.PagesDir,
		Npm:           b.config.Npm,
		ImportMap:     b.config.ImportMap,
	})

	if err != nil {
		return err
	}

	for _, page := range b.pages {
		if bundle, ok := bundles[page.id]; ok {
			var scriptTags strings.Builder
			var linkTags strings.Builder

			for _, src := range bundle.Scripts {
				scriptTags.WriteString(fmt.Sprintf(`<script type="module" src="%s"></script>`, src))
				scriptTags.WriteByte('\n')
			}

			for _, href := range bundle.Styles {
				linkTags.WriteString(fmt.Sprintf(`<link rel="stylesheet" href="%s">`, href))
				linkTags.WriteByte('\n')
			}

			page.Contents = strings.Replace(page.Contents, "</head>", linkTags.String()+"</head>", 1)
			page.Contents = strings.Replace(page.Contents, "</body>", scriptTags.String()+"</body>", 1)
		}
	}

	return nil
}

// Injects livereload scripts into pages.
func (b *Builder) injectDevScripts() {
	script := fmt.Sprintf("<script>%s</script>", livereload.JS)
	for _, page := range b.pages {
		page.Contents = strings.Replace(page.Contents, "</body>", script+"</body>", 1)
	}
}

// Minifies the contents of all pages concurrently
func (b *Builder) minifyPages() error {
	var g errgroup.Group
	for _, page := range b.pages {
		p := page
		g.Go(func() error {
			return b.minifyPage(p)
		})
	}
	return g.Wait()
}

// Minifies the contents of a single page.
func (b *Builder) minifyPage(p *Page) error {
	for _, p := range b.pages {
		html, err := b.minifier.String("text/html", p.Contents)
		if err != nil {
			return err
		}
		p.Contents = html
	}
	return nil
}

// Writes all files in the site into the output directory.
func (b *Builder) writeFiles() error {
	for _, page := range b.pages {
		dir := path.Dir(page.outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(page.outputPath, []byte(page.Contents), 0755); err != nil {
			return err
		}
	}

	for src, url := range b.assets {
		dst := path.Join(b.OutDir, url)
		copyFile(src, dst)
	}

	return nil
}
