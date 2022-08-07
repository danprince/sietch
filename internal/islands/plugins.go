package islands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

type virtualModulesConfig struct {
	filter  string
	modules map[string]api.OnLoadResult
}

func virtualModulesPlugin(c virtualModulesConfig) api.Plugin {
	namespace := "virtual"
	filter := c.filter

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
				return c.modules[args.Path], nil
			})
		},
	}
}

type islandsPluginOptions struct {
	resolveDir string
	frameworks []*Framework
}

// Plugin that transforms
func islandsFrameworkPlugin(opts islandsPluginOptions) api.Plugin {
	filter := `\?(static|hydrate)`
	pattern := regexp.MustCompile(filter)
	namespace := "islands"

	return api.Plugin{
		Name: "islands-framework-plugin",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{
				Filter: filter,
			}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				matches := pattern.FindStringSubmatch(args.Path)
				suffix := matches[0]
				importUrl := strings.TrimSuffix(args.Path, suffix)
				return api.OnResolveResult{
					Path:      importUrl,
					Suffix:    suffix,
					Namespace: namespace,
				}, nil
			})

			build.OnLoad(api.OnLoadOptions{
				Filter:    `.*`,
				Namespace: namespace,
			}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				// Use esbuild to resolve the actual name of the file, from the path
				// here (might have no extension, or be mapped with tsconfig etc).
				result := build.Resolve(args.Path, api.ResolveOptions{ResolveDir: opts.resolveDir})

				if len(result.Errors) > 0 {
					return api.OnLoadResult{Errors: result.Errors}, nil
				}

				framework, err := detectFramework(opts.frameworks, result.Path)

				if err != nil {
					return api.OnLoadResult{}, err
				}

				var contents string
				if args.Suffix == "?hydrate" {
					contents = framework.clientEntry(args.Path)
				} else {
					contents = framework.staticEntry(args.Path)
				}

				return api.OnLoadResult{
					Contents:   &contents,
					Loader:     api.LoaderJS,
					ResolveDir: opts.resolveDir,
				}, nil
			})
		},
	}
}

func importMapPlugin(importMap map[string]string) api.Plugin {
	filters := []string{}

	for name := range importMap {
		name = strings.ReplaceAll(name, `/`, `\/`)
		filters = append(filters, fmt.Sprintf(`^%s$`, name))
	}

	filter := "(" + strings.Join(filters, "|") + ")"

	return api.Plugin{
		Name: "import-maps",
		Setup: func(build api.PluginBuild) {
			// No need to register resolvers/loaders if the import map is empty
			// (it will be by default).
			if len(importMap) == 0 {
				return
			}

			build.OnResolve(api.OnResolveOptions{
				Filter: filter,
			}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				mapped := importMap[args.Path]
				resolve := build.Resolve(mapped, api.ResolveOptions{})

				if len(resolve.Errors) > 0 {
					return api.OnResolveResult{Errors: resolve.Errors}, nil
				}

				return api.OnResolveResult{
					Path:      resolve.Path,
					Namespace: resolve.Namespace,
				}, nil
			})
		},
	}
}
