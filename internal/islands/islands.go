package islands

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/danprince/sietch/internal/errors"
	"github.com/danprince/sietch/internal/islands/cdn"
	"github.com/evanw/esbuild/pkg/api"
	"rogchap.com/v8go"
)

type HydrationType uint8

const (
	Static HydrationType = iota
	HydrateOnLoad
	HydrateOnVisible
	HydrateOnIdle
)

type Island struct {
	Id         string
	Type       HydrationType
	Props      Props
	EntryPoint string
	ClientOnly bool
}

// Helper for templates that turns an island into HTML.
func (i *Island) String() string {
	if i.Type == Static {
		return i.Marker()
	} else if i.ClientOnly {
		return fmt.Sprintf(`<div id="%s"></div>`, i.Id)
	} else {
		return fmt.Sprintf(`<div id="%s">%s</div>`, i.Id, i.Marker())
	}
}

func (i *Island) Marker() string {
	return fmt.Sprintf("<!-- %s -->", i.Id)
}

// Distinct type for props that stringifies to JSON.
type Props map[string]any

func (p Props) String() string {
	data, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return string(data)
}

var (
	//go:embed client/runtime.ts
	sietchRuntimeSrc string

	// It's significantly faster to have one isolate than to create a context for
	// each evaluation.
	iso = v8go.NewIsolate()
)

type RenderOptions struct {
	ResolveDir string
	AssetsDir  string
	Frameworks []*Framework
	Islands    []*Island
	Npm        bool
	ImportMap  map[string]string
}

// Render the islands which produce static HTML into their respective pages.
func Render(opts RenderOptions) (map[string]string, error) {
	sourceFile := "sietch:static"
	code := strings.Builder{}

	for _, island := range opts.Islands {
		if !island.ClientOnly {
			code.WriteString(fmt.Sprintf("import { render as $r%s } from '%s?static';\n", island.Id, island.EntryPoint))
			code.WriteString(fmt.Sprintf("$elements['%s'] = $r%s(%s);", island.Id, island.Id, island.Props))
		}
	}

	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   code.String(),
			Sourcefile: sourceFile,
			Loader:     api.LoaderJS,
			ResolveDir: opts.ResolveDir,
		},
		Bundle:          true,
		Write:           false,
		Outdir:          opts.AssetsDir,
		Platform:        api.PlatformNeutral,
		Format:          api.FormatIIFE,
		Sourcemap:       api.SourceMapExternal,
		Target:          api.ES2021,
		JSXMode:         api.JSXModeAutomatic,
		JSXImportSource: Preact.jsxImportSource,
		Plugins: []api.Plugin{
			importMapPlugin(opts.ImportMap),
			cdn.Plugin(!opts.Npm),
			islandsFrameworkPlugin(islandsPluginOptions{
				resolveDir: opts.ResolveDir,
				frameworks: opts.Frameworks,
			}),
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

	script := fmt.Sprintf("globalThis.$elements = {};\n%s\n$elements", string(source))

	ctx := v8go.NewContext(iso)
	val, err := ctx.RunScript(script, sourceFile)

	if err != nil {
		return map[string]string{}, errors.V8Error(err, sourceFile, source, sourceMap, opts.AssetsDir)
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
	Frameworks    []*Framework
	IslandsByPage map[string][]*Island
	Production    bool
	OutDir        string
	AssetsDir     string
	ResolveDir    string
	Npm           bool
	ImportMap     map[string]string
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
	pagesModules := map[string]api.OnLoadResult{}

	for pageId, islands := range opts.IslandsByPage {
		virtualName := fmt.Sprintf(`page:%s`, pageId)

		bundles[pageId] = &BundleResult{}

		entryPoints = append(entryPoints, api.EntryPoint{
			InputPath:  virtualName,
			OutputPath: pageId,
		})

		sb := strings.Builder{}
		sb.WriteString("import { onIdle, onVisible } from 'sietch:runtime';\n")

		for _, island := range islands {
			id := island.Id
			props := island.Props
			src := island.EntryPoint
			el := fmt.Sprintf(`document.getElementById('%s')`, id)

			switch island.Type {
			case HydrateOnLoad:
				sb.WriteString(fmt.Sprintf("import { hydrate as $h%s } from '%s?hydrate';\n", id, src))
				sb.WriteString(fmt.Sprintf("$h%s(%s, %s);\n", id, props, el))
			case HydrateOnIdle:
				sb.WriteString(fmt.Sprintf("onIdle().then(() => import('%s?hydrate'))", src))
				sb.WriteString(fmt.Sprintf(".then($c => $c.hydrate(%s, %s))\n", props, el))
			case HydrateOnVisible:
				sb.WriteString(fmt.Sprintf("onVisible(%s).then(() => import('%s?hydrate'))", el, src))
				sb.WriteString(fmt.Sprintf(".then($c => $c.hydrate(%s, %s))\n", props, el))
			}
		}

		contents := sb.String()
		pagesModules[virtualName] = api.OnLoadResult{
			Contents:   &contents,
			ResolveDir: opts.ResolveDir,
			Loader:     api.LoaderJS,
		}
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
		JSXImportSource:   Preact.jsxImportSource,
		Plugins: []api.Plugin{
			importMapPlugin(opts.ImportMap),
			cdn.Plugin(!opts.Npm),
			islandsFrameworkPlugin(islandsPluginOptions{
				frameworks: opts.Frameworks,
				resolveDir: opts.ResolveDir,
			}),
			virtualModulesPlugin(virtualModulesConfig{
				filter:  `^page:`,
				modules: pagesModules,
			}),
			virtualModulesPlugin(virtualModulesConfig{
				filter: `^sietch:`,
				modules: map[string]api.OnLoadResult{
					"sietch:runtime": {
						Contents: &sietchRuntimeSrc,
						Loader:   api.LoaderTS,
					},
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
