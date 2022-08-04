package builder

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
	"sync"
	"text/template"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/alecthomas/chroma/styles"
	"github.com/danprince/sietch/internal/errors"
	"github.com/danprince/sietch/internal/islands"
	"github.com/danprince/sietch/internal/livereload"
	"github.com/danprince/sietch/internal/mdext"
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

	frameworkMap = map[string]islands.Framework{
		islands.Vanilla.Id:      islands.Vanilla,
		islands.Preact.Id:       islands.Preact,
		islands.PreactRemote.Id: islands.PreactRemote,
	}
)

// Manages the state of the site throughout the duration of the process.
type Builder struct {
	RootDir      string
	PagesDir     string
	AssetsDir    string
	OutDir       string
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
	framework    islands.Framework
}

type Config struct {
	SyntaxColor string
	Framework   string
	DateFormat  string
}

var defaultConfig = Config{
	SyntaxColor: "algol_nu",
	Framework:   "vanilla",
	DateFormat:  "2006-1-2",
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
	return &Builder{
		Mode:         mode,
		RootDir:      dir,
		PagesDir:     dir,
		OutDir:       path.Join(dir, "_site"),
		AssetsDir:    path.Join(dir, "_site/_assets"),
		templateFile: path.Join(dir, "_template.html"),
		configFile:   path.Join(dir, ".sietch.json"),
		config:       defaultConfig,
		pages:        []*Page{},
		assets:       map[string]string{},
		assetsMu:     sync.Mutex{},
		index:        map[string][]*Page{},
		framework:    islands.Vanilla,
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

	if b.Mode == Development {
		b.injectDevScripts()
	}

	err = b.writeFiles()
	if err != nil {
		return err
	}

	return nil
}

// Read and parse the site's global page template.
func (b *Builder) readConfig() error {
	contents, err := os.ReadFile(b.configFile)

	if os.IsNotExist(err) {
		return nil
	}

	err = json.Unmarshal(contents, &b.config)

	if err != nil {
		return errors.JsonParseError(err, b.configFile, string(contents))
	}

	return nil
}

// Configure everything required to start building.
func (b *Builder) applyConfig() error {
	if _, ok := frameworkMap[b.config.Framework]; !ok {
		allowed := []string{}

		for s := range frameworkMap {
			allowed = append(allowed, s)
		}

		return errors.ConfigError{
			File:    b.configFile,
			Key:     "Framework",
			Value:   b.config.Framework,
			Allowed: allowed,
		}
	}

	if _, ok := styles.Registry[b.config.SyntaxColor]; !ok {
		allowed := []string{}

		for s := range styles.Registry {
			allowed = append(allowed, s)
		}

		return errors.ConfigError{
			File:    b.configFile,
			Key:     "SyntaxColor",
			Value:   b.config.SyntaxColor,
			Allowed: allowed,
		}
	}

	b.framework = frameworkMap[b.config.Framework]

	b.markdown = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			mdext.ExternalLinks,
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
		"props": func(kvs ...any) islands.Props {
			p := make(islands.Props, len(kvs)/2)

			for i := 0; i < len(kvs)-1; i++ {
				k := kvs[i]
				v := kvs[i+1]
				if key, ok := k.(string); ok {
					p[key] = v
				}
			}

			return p
		},
		"render": func(entryPoint string, props map[string]any) *islands.Island {
			if entryPoint[0] == '.' {
				entryPoint = "." + path.Join(page.Dir, entryPoint)
			}

			return page.addIsland(entryPoint, props)
		},
		"clientOnLoad": func(island *islands.Island) *islands.Island {
			island.Type = islands.ClientOnLoad
			return island
		},
		"clientOnVisible": func(island *islands.Island) *islands.Island {
			island.Type = islands.ClientOnVisible
			return island
		},
		"clientOnIdle": func(island *islands.Island) *islands.Island {
			island.Type = islands.ClientOnIdle
			return island
		},
		"clientOnly": func(island *islands.Island) *islands.Island {
			island.ClientOnly = true
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
		return errors.Wrap("template", err)
	}

	t, err := template.New("template").Funcs(funcs).Parse(string(contents))

	if err != nil {
		return errors.TemplateParseError(err, b.templateFile, string(contents), 0)
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
		return errors.TemplateParseError(err, b.templateFile, string(contents), page.contentStartLine)
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
	globalTemplate, err := b.template.Funcs(funcs).Parse("")

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
		Framework:  b.framework,
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
		Framework:     b.framework,
		IslandsByPage: islandsByPage,
		Production:    b.Mode == Production,
		OutDir:        b.OutDir,
		AssetsDir:     b.AssetsDir,
		ResolveDir:    b.PagesDir,
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

	for src, url := range b.assets {
		dst := path.Join(b.OutDir, url)
		dir := path.Dir(dst)

		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

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
	}

	return nil
}
