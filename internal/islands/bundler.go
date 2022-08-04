package islands

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/danprince/sietch/internal/errors"
	"github.com/evanw/esbuild/pkg/api"
	"rogchap.com/v8go"
)

var (
	//go:embed client/sietch-client.ts
	sietchClientSrc string

	// It's significantly faster to have one isolate than to create a context for
	// each evaluation.
	iso = v8go.NewIsolate()
)

type RenderOptions struct {
	ResolveDir string
	AssetsDir  string
	Framework  Framework
	Islands    []*Island
}

// Render the islands which produce static HTML into their respective pages.
func Render(opts RenderOptions) (map[string]string, error) {
	code := opts.Framework.staticEntryPoint(opts.Islands)

	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   code,
			Sourcefile: "server-entry.js",
			ResolveDir: opts.ResolveDir,
		},
		Bundle:          true,
		Write:           false,
		Outdir:          opts.AssetsDir,
		Platform:        api.PlatformNeutral,
		Format:          api.FormatIIFE,
		Sourcemap:       api.SourceMapExternal,
		JSXMode:         api.JSXModeAutomatic,
		JSXImportSource: "preact",
		Plugins: []api.Plugin{
			importMapPlugin(opts.Framework.importMap),
			httpImportsPlugin(),
		},
	})

	if len(result.Errors) > 0 {
		return map[string]string{}, errors.EsbuildError(result)
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

	name := "server-entry.js"
	script := fmt.Sprintf("globalThis.$elements = {};\n%s\n$elements", string(source))

	ctx := v8go.NewContext(iso)
	val, err := ctx.RunScript(script, name)

	if err != nil {
		return map[string]string{}, errors.V8Error(err, name, source, sourceMap, opts.AssetsDir)
	}

	s, err := v8go.JSONStringify(ctx, val)

	if err != nil {
		return map[string]string{}, err
	}

	elements := map[string]string{}
	err = json.Unmarshal([]byte(s), &elements)

	if err != nil {
		return map[string]string{}, err
	}

	return elements, nil
}

type BundleOptions struct {
	Framework     Framework
	IslandsByPage map[string][]*Island
	Production    bool
	OutDir        string
	AssetsDir     string
	ResolveDir    string
}

type BundleResult struct {
	Styles  []string
	Scripts []string
}

// Create client side bundles for the dynamic islands and inject their scripts
// and styles into pages as necessary.
func Bundle(opts BundleOptions) (map[string]*BundleResult, error) {
	bundles := map[string]*BundleResult{}
	entryPoints := []api.EntryPoint{}

	for pageId := range opts.IslandsByPage {
		bundles[pageId] = &BundleResult{}

		entryPoints = append(entryPoints, api.EntryPoint{
			InputPath:  fmt.Sprintf("%s?browser", pageId),
			OutputPath: pageId,
		})
	}

	entryNames := "bundle-[name]-[hash]"
	chunkNames := "chunk-[hash]"
	assetNames := "media/[name]-[hash]"

	if !opts.Production {
		// Remove hashes in development to prevent ending up with hundreds of
		// versions of the file in the assets dir.
		entryNames = "bundle-[name]"
	}

	result := api.Build(api.BuildOptions{
		EntryPointsAdvanced: entryPoints,

		EntryNames:        entryNames,
		ChunkNames:        chunkNames,
		AssetNames:        assetNames,
		Bundle:            true,
		Write:             true,
		Splitting:         true,
		Outdir:            opts.AssetsDir,
		Platform:          api.PlatformBrowser,
		Sourcemap:         api.SourceMapLinked,
		Format:            api.FormatESModule,
		MinifyWhitespace:  opts.Production,
		MinifySyntax:      opts.Production,
		MinifyIdentifiers: opts.Production,
		JSXMode:           api.JSXModeAutomatic,
		JSXImportSource:   "preact",
		Plugins: []api.Plugin{
			browserPagesPlugin(opts),
			importMapPlugin(opts.Framework.importMap),
			httpImportsPlugin(),
			virtualModulesPlugin(map[string]api.OnLoadResult{
				"@sietch/client": {
					Contents: &sietchClientSrc,
					Loader:   api.LoaderTS,
				},
			}),
		},
	})

	if len(result.Errors) > 0 {
		return bundles, errors.EsbuildError(result)
	}

	pageIdPattern := regexp.MustCompile(`bundle-(\w+)`)

	for _, file := range result.OutputFiles {
		matches := pageIdPattern.FindStringSubmatch(file.Path)

		if matches == nil {
			continue
		}

		pageId := matches[1]
		bundle := bundles[pageId]
		href := strings.TrimPrefix(file.Path, opts.OutDir)

		switch path.Ext(file.Path) {
		case ".js":
			bundle.Scripts = append(bundle.Scripts, href)
		case ".css":
			bundle.Styles = append(bundle.Styles, href)
		}
	}

	return bundles, nil
}
