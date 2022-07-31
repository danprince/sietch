package islands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	esbuild "github.com/evanw/esbuild/pkg/api"
	"rogchap.com/v8go"
)

var iso = v8go.NewIsolate()

type Element struct {
	id         string
	entryPoint string
	marker     string
	Props      map[string]any
	CSR        bool
	SSR        bool
}

type Ctx struct {
	ResolveDir string
	Elements   map[string]*Element
}

type Framework struct {
	importMap           map[string]string
	jsxImportSource     string
	createRenderScript  func(ctx *Ctx) (string, error)
	createHydrateScript func(ctx *Ctx) (string, error)
}

var loader = map[string]esbuild.Loader{
	".aac":         esbuild.LoaderFile,
	".css":         esbuild.LoaderFile,
	".eot":         esbuild.LoaderFile,
	".flac":        esbuild.LoaderFile,
	".gif":         esbuild.LoaderFile,
	".ico":         esbuild.LoaderFile,
	".jpeg":        esbuild.LoaderFile,
	".jpg":         esbuild.LoaderFile,
	".js":          esbuild.LoaderJS,
	".jsx":         esbuild.LoaderJSX,
	".json":        esbuild.LoaderJSON,
	".mp3":         esbuild.LoaderFile,
	".mp4":         esbuild.LoaderFile,
	".ogg":         esbuild.LoaderFile,
	".otf":         esbuild.LoaderFile,
	".png":         esbuild.LoaderFile,
	".svg":         esbuild.LoaderFile,
	".ts":          esbuild.LoaderTS,
	".tsx":         esbuild.LoaderTSX,
	".ttf":         esbuild.LoaderFile,
	".wav":         esbuild.LoaderFile,
	".webm":        esbuild.LoaderFile,
	".webmanifest": esbuild.LoaderFile,
	".webp":        esbuild.LoaderFile,
	".woff":        esbuild.LoaderFile,
	".woff2":       esbuild.LoaderFile,
}

func NewContext(resolveDir string) Ctx {
	return Ctx{
		ResolveDir: resolveDir,
		Elements:   map[string]*Element{},
	}
}

func (ctx *Ctx) needsSSR() bool {
	for _, el := range ctx.Elements {
		if el.SSR {
			return true
		}
	}
	return false
}

func (ctx *Ctx) needsCSR() bool {
	for _, el := range ctx.Elements {
		if el.CSR {
			return true
		}
	}
	return false
}

func (ctx *Ctx) AddElement(entryPoint string, props map[string]any) *Element {
	num := len(ctx.Elements)
	id := fmt.Sprintf("$h%d", num)
	marker := fmt.Sprintf("<!-- %s -->", id)
	element := &Element{
		id:         id,
		entryPoint: entryPoint,
		marker:     marker,
		Props:      props,
		SSR:        true,
		CSR:        false,
	}
	ctx.Elements[id] = element
	return element
}

func (e *Element) String() string {
	return fmt.Sprintf(`<div id="%s">%s</div>`, e.id, e.marker)
}

type BundleOptions struct {
	Framework  *Framework
	OutDir     string
	PublicPath string
	Production bool
}

type RuntimeResult struct {
	Scripts []string
	Links   []string
}

func (ctx *Ctx) CreateRuntime(options *BundleOptions) (RuntimeResult, error) {
	script, err := options.Framework.createHydrateScript(ctx)

	if err != nil {
		return RuntimeResult{}, err
	}

	entryNames := "[dir]/[name]"

	if options.Production {
		entryNames = "[hash]"
	}

	result := esbuild.Build(esbuild.BuildOptions{
		EntryPoints:       []string{"@sietch/client"},
		Write:             true,
		Bundle:            true,
		EntryNames:        entryNames,
		Outdir:            options.OutDir,
		MinifySyntax:      options.Production,
		MinifyWhitespace:  options.Production,
		MinifyIdentifiers: options.Production,
		Sourcemap:         esbuild.SourceMapLinked,
		Platform:          esbuild.PlatformBrowser,
		Format:            esbuild.FormatESModule,
		JSXMode:           esbuild.JSXModeAutomatic,
		JSXImportSource:   options.Framework.jsxImportSource,
		PublicPath:        options.PublicPath,
		Loader:            loader,
		Plugins: []esbuild.Plugin{
			virtualModulesPlugin(map[string]virtualModule{
				"@sietch/client": {
					contents:   &script,
					resolveDir: ctx.ResolveDir,
					loader:     esbuild.LoaderJS,
				},
			}),
			httpImportsPlugin(options.Framework.importMap),
		},
	})

	for _, err := range result.Errors {
		// TODO: Use better errors
		fmt.Println("CLIENT BUNDLE ERROR", err)
	}

	for _, err := range result.Errors {
		// TODO: Use better errors
		return RuntimeResult{}, errors.New(err.Text)
	}

	var scripts []string
	var links []string

	for _, file := range result.OutputFiles {
		src := file.Path

		switch path.Ext(src) {
		case ".js":
			scripts = append(scripts, src)
		case ".css":
			links = append(links, src)
		}
	}

	// Prevent any JavaScript making it down to the client if no elements
	// requested a client side render.
	if !ctx.needsCSR() {
		scripts = []string{}
	}

	return RuntimeResult{
		Scripts: scripts,
		Links:   links,
	}, nil
}

func (ctx *Ctx) CreateStaticHtml(options *BundleOptions) (map[string]string, error) {
	if !ctx.needsSSR() {
		return map[string]string{}, nil
	}

	code, err := ctx.staticBundle(options)

	if err != nil {
		return nil, err
	}

	staticHtml, err := ctx.runInV8(code)

	if err != nil {
		return nil, err
	}

	return staticHtml, nil
}

func (ctx *Ctx) staticBundle(options *BundleOptions) (string, error) {
	script, err := options.Framework.createRenderScript(ctx)

	if err != nil {
		return "", err
	}

	result := esbuild.Build(esbuild.BuildOptions{
		Stdin: &esbuild.StdinOptions{
			Contents:   script,
			Loader:     esbuild.LoaderJS,
			ResolveDir: ctx.ResolveDir,
		},
		Write:           false,
		Bundle:          true,
		Outdir:          options.OutDir,
		Sourcemap:       esbuild.SourceMapInline,
		Platform:        esbuild.PlatformNeutral,
		Format:          esbuild.FormatIIFE,
		JSXMode:         esbuild.JSXModeAutomatic,
		JSXImportSource: options.Framework.jsxImportSource,
		Loader:          loader,
		PublicPath:      options.PublicPath,
		Plugins: []esbuild.Plugin{
			httpImportsPlugin(options.Framework.importMap),
		},
	})

	for _, err := range result.Errors {
		// TODO: Use better errors
		fmt.Println("STATIC BUNDLE ERROR", err)
	}

	for _, err := range result.Errors {
		// TODO: Use better errors
		return "", errors.New(err.Text)
	}

	var outfile esbuild.OutputFile

	// Make sure we pick the correct file from the outputs, sometimes there
	// will be other assets here (e.g. images).
	for _, file := range result.OutputFiles {
		if strings.HasSuffix(file.Path, "stdin.js") {
			outfile = file
		}
	}

	// The static bundle that esbuild produces renders each of the elements into
	// a global variable called `$`. A reference to `$` needs to be the final
	// thing in this script so that the script evaluates to the correct value.
	js := string(outfile.Contents) + ";$"

	return js, nil
}

func (ctx *Ctx) runInV8(js string) (map[string]string, error) {
	v8Ctx := v8go.NewContext(iso)
	defer v8Ctx.Close()

	val, err := v8Ctx.RunScript(js, "hydrate.js")

	if err != nil {
		// TODO: Wrap with a meaningful error, incl. stack trace
		tmp := path.Join(os.TempDir(), "/v8debug.js")
		os.WriteFile(tmp, []byte(js), os.ModePerm)
		fmt.Println("V8 ERROR", err, tmp)
		return nil, err
	}

	str, err := v8go.JSONStringify(v8Ctx, val)
	if err != nil {
		return nil, err
	}

	var staticHtml map[string]string

	err = json.Unmarshal([]byte(str), &staticHtml)
	if err != nil {
		return nil, err
	}

	return staticHtml, nil
}
