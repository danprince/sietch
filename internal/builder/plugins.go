package builder

import (
	"fmt"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

func browserPagesPlugin(b *Builder) api.Plugin {
	namespace := "browser-pages"
	filter := `\?browser`

	pagesById := map[string]*Page{}

	for _, p := range b.pages {
		pagesById[p.id] = p
	}

	return api.Plugin{
		Name: "browser-pages-plugin",
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
				pageId := strings.ReplaceAll(args.Path, "?browser", "")
				page := pagesById[pageId]
				contents := b.framework.clientEntryPoint(page.islands)
				return api.OnLoadResult{
					Contents:   &contents,
					Loader:     api.LoaderJS,
					ResolveDir: b.PagesDir,
				}, nil
			})
		},
	}
}

func virtualModulesPlugin(modules map[string]api.OnLoadResult) api.Plugin {
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
				return modules[args.Path], nil
			})
		},
	}
}
