package islands

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
)

var httpImportNamespace = "http-import"

type virtualModule struct {
	contents   *string
	resolveDir string
	loader     api.Loader
}

func virtualModulesPlugin(modules map[string]virtualModule) api.Plugin {
	namespace := "virtual-modules"
	names := []string{}

	for name := range modules {
		// regexp escapes
		name = strings.ReplaceAll(name, "/", `\/`)
		names = append(names, name)
	}

	filter := fmt.Sprintf(`^(%s)$`, strings.Join(names, "|"))

	return api.Plugin{
		Name: "virtual-modules-plugin",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{
				Filter: filter,
			}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				return api.OnResolveResult{
					Path:      args.Path,
					Namespace: namespace,
				}, nil
			})

			build.OnLoad(api.OnLoadOptions{
				Filter:    `.*`,
				Namespace: namespace,
			}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				module := modules[args.Path]
				return api.OnLoadResult{
					Contents:   module.contents,
					ResolveDir: module.resolveDir,
					Loader:     module.loader,
				}, nil
			})
		},
	}
}

func httpImportsPlugin(importMap map[string]string) api.Plugin {
	var names []string

	for name := range importMap {
		names = append(names, name)
	}

	modulesFilter := fmt.Sprintf(`^(%s)$`, strings.Join(names, "|"))

	return api.Plugin{
		Name: "esbuild-http-import-plugin",
		Setup: func(build api.PluginBuild) {
			// Resolve http imports from modules in the import map
			build.OnResolve(api.OnResolveOptions{
				Filter: modulesFilter,
			}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				return api.OnResolveResult{
					Path:      importMap[args.Path],
					Namespace: httpImportNamespace,
				}, nil
			})

			// Add the http-import namespace to non-mapped imports too
			build.OnResolve(api.OnResolveOptions{
				Filter: `^https?://`,
			}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				return api.OnResolveResult{
					Path:      args.Path,
					Namespace: httpImportNamespace,
				}, nil
			})

			// Resolve urls from inside downloaded files
			build.OnResolve(api.OnResolveOptions{
				Filter:    `.*`,
				Namespace: httpImportNamespace,
			}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				base, err := url.Parse(args.Importer)

				if err != nil {
					return api.OnResolveResult{}, err
				}

				relative, err := url.Parse(args.Path)

				if err != nil {
					return api.OnResolveResult{}, err
				}

				return api.OnResolveResult{
					Path:      base.ResolveReference(relative).String(),
					Namespace: httpImportNamespace,
				}, nil
			})

			// Load the module over http.
			build.OnLoad(api.OnLoadOptions{
				Filter:    `.*`,
				Namespace: httpImportNamespace,
			}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				contents, err := downloadWithCache(args.Path)

				if err != nil {
					return api.OnLoadResult{}, err
				}

				return api.OnLoadResult{
					Contents:   &contents,
					ResolveDir: "/tmp",
				}, nil
			})
		},
	}
}

var cacheDir = path.Join(os.TempDir(), ".sietch/http-imports")
var cachedModules = map[string]string{}
var cachedModulesLock sync.Mutex

func init() {
	os.MkdirAll(cacheDir, os.ModePerm)
	dirents, _ := os.ReadDir(cacheDir)
	for _, dirent := range dirents {
		if !dirent.IsDir() {
			name, _ := url.QueryUnescape(dirent.Name())
			contents, _ := os.ReadFile(path.Join(cacheDir, dirent.Name()))
			cachedModules[name] = string(contents)
		}
	}
}

func downloadWithCache(href string) (string, error) {
	cachedModulesLock.Lock()
	mod, ok := cachedModules[href]
	cachedModulesLock.Unlock()

	if ok {
		return mod, nil
	}

	res, err := http.Get(href)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	out, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	contents := string(out)

	cachedModulesLock.Lock()
	cachedModules[href] = contents
	name := url.QueryEscape(href)
	os.WriteFile(path.Join(cacheDir, name), []byte(contents), os.ModePerm)
	cachedModulesLock.Unlock()

	return contents, nil
}
