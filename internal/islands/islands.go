package islands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"

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
	createRenderScript  func(ctx *Ctx) (string, error)
	createHydrateScript func(ctx *Ctx) (string, error)
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

func (ctx *Ctx) CreateStaticHtml(framework *Framework) (map[string]string, error) {
	if !ctx.needsSSR() {
		return map[string]string{}, nil
	}

	code, err := ctx.staticBundle(framework)

	if err != nil {
		return nil, err
	}

	staticHtml, err := ctx.runInV8(code)

	if err != nil {
		return nil, err
	}

	return staticHtml, nil
}

type runtime struct {
	Scripts []string
	Links   []string
}

type RuntimeOptions struct {
	Framework  *Framework
	OutDir     string
	Production bool
}

func (ctx *Ctx) CreateRuntime(options RuntimeOptions) (runtime, error) {
	script, err := options.Framework.createHydrateScript(ctx)

	if err != nil {
		return runtime{}, err
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
		Sourcemap:         esbuild.SourceMapExternal,
		Platform:          esbuild.PlatformBrowser,
		Format:            esbuild.FormatESModule,
		JSXMode:           esbuild.JSXModeTransform,
		Plugins: []esbuild.Plugin{
			dynamicEntryPlugin(dynamicEntryPoint{
				name:       "@sietch/client",
				contents:   &script,
				resolveDir: ctx.ResolveDir,
				loader:     esbuild.LoaderJS,
			}),
			httpExternalsPlugin(options.Framework.importMap),
		},
	})

	for _, err := range result.Errors {
		// TODO: Use better errors
		fmt.Println("CLIENT BUNDLE ERROR", err)
	}

	for _, err := range result.Errors {
		// TODO: Use better errors
		return runtime{}, errors.New(err.Text)
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

	return runtime{
		Scripts: scripts,
		Links:   links,
	}, nil
}

func (ctx *Ctx) staticBundle(framework *Framework) (string, error) {
	script, err := framework.createRenderScript(ctx)

	if err != nil {
		return "", err
	}

	result := esbuild.Build(esbuild.BuildOptions{
		Stdin: &esbuild.StdinOptions{
			Contents:   script,
			Loader:     esbuild.LoaderJS,
			ResolveDir: ctx.ResolveDir,
		},
		Write:     false,
		Outdir:    "/tmp", // no-op because `Write` is false but still required for asset imports
		Bundle:    true,
		Sourcemap: esbuild.SourceMapInline,
		Platform:  esbuild.PlatformNeutral,
		Format:    esbuild.FormatIIFE,
		JSXMode:   esbuild.JSXModeTransform,
		Plugins: []esbuild.Plugin{
			httpImportMapPlugin(framework.importMap),
			httpImportsPlugin(),
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

	// The static bundle that esbuild produces renders each of the elements into
	// a global variable called `$`. A reference to `$` needs to be the final
	// thing in this script so that the script evaluates to the correct value.
	js := string(result.OutputFiles[0].Contents) + ";$"

	return js, nil
}

func (ctx *Ctx) runInV8(js string) (map[string]string, error) {
	v8Ctx := v8go.NewContext(iso)
	defer v8Ctx.Close()

	os.WriteFile("/tmp/test.js", []byte(js), os.ModePerm)
	val, err := v8Ctx.RunScript(js, "hydrate.js")
	// TODO: Wrap with a meaningful error, incl. stack trace
	if err != nil {
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
