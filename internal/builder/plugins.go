package builder

import (
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
				contents := b.framework.clientEntryPoint(b, page)
				return api.OnLoadResult{
					Contents:   &contents,
					Loader:     api.LoaderJS,
					ResolveDir: b.PagesDir,
				}, nil
			})
		},
	}
}
