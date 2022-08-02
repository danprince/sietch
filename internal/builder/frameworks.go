package builder

import (
	"encoding/json"
	"fmt"
	"strings"
)

var Vanilla = Framework{
	staticEntryPoint: func(builder *Builder) string {
		var b strings.Builder
		for _, page := range builder.pages {
			for _, island := range page.islands {
				if island.Type != IslandClientOnly {
					props, _ := json.Marshal(island.Props)
					b.WriteString(fmt.Sprintf("import { render as $r%s } from '%s';\n", island.Id, island.EntryPoint))
					b.WriteString(fmt.Sprintf("$elements['%s'] = $r%s(%s);\n", island.Id, island.Id, props))
				}
			}
		}
		return b.String()
	},
	clientEntryPoint: func(builder *Builder, page *Page) string {
		var b strings.Builder
		for _, island := range page.islands {
			if island.Type != IslandStatic {
				props, _ := json.Marshal(island.Props)
				elem := fmt.Sprintf("document.getElementById('%s')", island.Id)
				b.WriteString(fmt.Sprintf("import { hydrate as $h%s } from '%s';\n", island.Id, island.EntryPoint))
				b.WriteString(fmt.Sprintf("$h%s(%s, %s);\n", island.Id, props, elem))
			}
		}
		return b.String()
	},
}
