package cdn

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

const (
	cdnUrl = "https://esm.sh"
)

func Plugin(enabled bool) api.Plugin {
	namespace := "cdn"
	filter := `^[a-z@]`

	return api.Plugin{
		Name: "cdn-imports-plugin",
		Setup: func(build api.PluginBuild) {
			// When this plugin is disabled, esbuild will resolve from node_modules
			if !enabled {
				return
			}

			// Add the cdn namespace to any bare module imports
			build.OnResolve(api.OnResolveOptions{
				Filter: filter,
			}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {

				// Never attempt to load modules with protocols (these are used for
				// internal modules e.g "sietch:runtime" and "page:1234").
				if strings.ContainsRune(args.Path, ':') {
					return api.OnResolveResult{}, nil
				}

				return api.OnResolveResult{
					Path:      fmt.Sprintf("%s/%s", cdnUrl, args.Path),
					Namespace: namespace,
				}, nil
			})

			// Ensure that absolute import paths from inside cdn imports are also
			// resolved back to their base url.
			build.OnResolve(api.OnResolveOptions{
				Filter:    `.*`,
				Namespace: namespace,
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
					Namespace: namespace,
				}, nil
			})

			build.OnLoad(api.OnLoadOptions{
				Filter:    `.*`,
				Namespace: namespace,
			}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				contents, err := downloadWithCache(args.Path)

				if err != nil {
					return api.OnLoadResult{}, err
				}

				return api.OnLoadResult{
					Contents:   &contents,
					ResolveDir: "/",
				}, nil
			})
		},
	}
}

var (
	cacheDir          = path.Join(os.TempDir(), ".sietch/http-imports")
	cachedModules     = map[string]string{}
	cachedModulesLock sync.Mutex
)

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

	fmt.Printf("downloading %s \n", href)

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
