package islands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"rogchap.com/v8go"
)

var loader = map[string]api.Loader{
	".aac":         api.LoaderFile,
	".css":         api.LoaderCSS,
	".eot":         api.LoaderFile,
	".flac":        api.LoaderFile,
	".gif":         api.LoaderFile,
	".ico":         api.LoaderFile,
	".jpeg":        api.LoaderFile,
	".jpg":         api.LoaderFile,
	".js":          api.LoaderJS,
	".jsx":         api.LoaderJSX,
	".json":        api.LoaderJSON,
	".mp3":         api.LoaderFile,
	".mp4":         api.LoaderFile,
	".ogg":         api.LoaderFile,
	".otf":         api.LoaderFile,
	".png":         api.LoaderFile,
	".svg":         api.LoaderFile,
	".ts":          api.LoaderTS,
	".tsx":         api.LoaderTSX,
	".ttf":         api.LoaderFile,
	".wav":         api.LoaderFile,
	".webm":        api.LoaderFile,
	".webmanifest": api.LoaderFile,
	".webp":        api.LoaderFile,
	".woff":        api.LoaderFile,
	".woff2":       api.LoaderFile,
}

type BundleOptions struct {
	Framework  *Framework
	OutDir     string
	PublicPath string
	Production bool
}

type Render struct {
	Elements map[string]string
	Scripts  []string
	Styles   []string
}

type ClientBundle struct {
	Scripts []string
	Styles  []string
}

func (ctx *Ctx) Build(options BundleOptions) (Render, error) {
	static, err := ctx.staticRender(&options)

	if err != nil {
		return Render{}, err
	}

	client, err := ctx.bundleClient(&options)

	if err != nil {
		return Render{}, err
	}

	return Render{
		Elements: static.Elements,
		Scripts:  append(client.Scripts, static.Scripts...),
		Styles:   append(client.Styles, static.Styles...),
	}, nil
}

func (ctx *Ctx) bundleClient(options *BundleOptions) (ClientBundle, error) {
	script, err := options.Framework.createHydrateScript(ctx)

	if err != nil {
		return ClientBundle{}, err
	}

	entryNames := "[dir]/[name]"

	if options.Production {
		entryNames = "[hash]"
	}

	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{"@sietch/client"},
		Write:             true,
		Bundle:            true,
		EntryNames:        entryNames,
		Outdir:            options.OutDir,
		MinifySyntax:      options.Production,
		MinifyWhitespace:  options.Production,
		MinifyIdentifiers: options.Production,
		Sourcemap:         api.SourceMapLinked,
		Platform:          api.PlatformBrowser,
		Format:            api.FormatESModule,
		JSXMode:           api.JSXModeAutomatic,
		JSXImportSource:   options.Framework.jsxImportSource,
		PublicPath:        options.PublicPath,
		Loader:            loader,
		Plugins: []api.Plugin{
			virtualModulesPlugin(map[string]virtualModule{
				"@sietch/client": {
					contents:   &script,
					resolveDir: ctx.ResolveDir,
					loader:     api.LoaderJS,
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
		return ClientBundle{}, errors.New(err.Text)
	}

	var scripts []string
	var styles []string

	for _, file := range result.OutputFiles {
		src := file.Path

		switch path.Ext(src) {
		case ".js":
			scripts = append(scripts, src)
		case ".css":
			styles = append(styles, src)
		}
	}

	// Prevent any JavaScript making it down to the client if no elements
	// requested a client side render.
	if !ctx.needsCSR() {
		scripts = []string{}
	}

	return ClientBundle{
		Scripts: scripts,
		Styles:  styles,
	}, nil
}

func (ctx *Ctx) staticRender(options *BundleOptions) (Render, error) {
	if !ctx.needsSSR() {
		return Render{}, nil
	}

	script, err := options.Framework.createRenderScript(ctx)

	if err != nil {
		return Render{}, err
	}

	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   script,
			Loader:     api.LoaderJS,
			ResolveDir: ctx.ResolveDir,
		},
		Write:           false,
		Bundle:          true,
		Outdir:          options.OutDir,
		Sourcemap:       api.SourceMapInline,
		Platform:        api.PlatformNeutral,
		Format:          api.FormatIIFE,
		JSXMode:         api.JSXModeAutomatic,
		JSXImportSource: options.Framework.jsxImportSource,
		Loader:          loader,
		PublicPath:      options.PublicPath,
		Plugins: []api.Plugin{
			httpImportsPlugin(options.Framework.importMap),
		},
	})

	for _, err := range result.Errors {
		// TODO: Use better errors
		fmt.Println("STATIC BUNDLE ERROR", err)
	}

	for _, err := range result.Errors {
		// TODO: Use better errors
		return Render{}, errors.New(err.Text)
	}

	var outfile api.OutputFile

	// Write file loader assets to disk to ensure they're available for
	// components that don't render at runtime.
	writeStaticOutputs(result)

	// Make sure we pick the correct file from the outputs, sometimes there
	// will be other assets here (e.g. images).
	for _, file := range result.OutputFiles {
		if strings.HasSuffix(file.Path, "stdin.js") {
			outfile = file
			break
		}
	}

	// The static bundle renders each of the elements into a global called `$`.
	// A reference to `$` needs to be the final thing in this script so that
	// the script evaluates to the correct value.
	js := string(outfile.Contents) + ";$"

	if err != nil {
		return Render{}, err
	}

	elements, err := ctx.runInV8(js)

	if err != nil {
		return Render{}, err
	}

	return Render{
		Elements: elements,
	}, nil
}

func writeStaticOutputs(result api.BuildResult) error {
	for _, file := range result.OutputFiles {
		ext := path.Ext(file.Path)
		l, ok := loader[ext]
		if ok && l == api.LoaderFile {
			err := os.MkdirAll(path.Dir(file.Path), 0755)
			if err != nil {
				return err
			}

			err = os.WriteFile(file.Path, []byte(file.Contents), 0644)
			if err != nil {
				return err
			}
		}
	}
	return nil
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
