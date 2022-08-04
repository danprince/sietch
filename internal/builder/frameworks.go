package builder

import (
	"encoding/json"
	"fmt"
	"strings"
)

var Vanilla = Framework{
	staticEntryPoint: func(islands []*Island) string {
		var b strings.Builder
		for _, island := range islands {
			props, _ := json.Marshal(island.Props)
			b.WriteString(fmt.Sprintf("import { render as $r%s } from '%s';\n", island.Id, island.EntryPoint))
			b.WriteString(fmt.Sprintf("$elements['%s'] = $r%s(%s);\n", island.Id, island.Id, props))
		}
		return b.String()
	},
	clientEntryPoint: func(islands []*Island) string {
		var b strings.Builder
		b.WriteString("import { onIdle, onVisible } from '@sietch/client'\n")

		for _, island := range islands {
			elem := fmt.Sprintf("document.getElementById('%s')", island.Id)
			props, _ := json.Marshal(island.Props)

			switch island.Type {
			case IslandClientOnLoad:
				b.WriteString(fmt.Sprintf(`
import { hydrate as $h%s } from '%s';
$h%s(%s, %s);
`, island.Id, island.EntryPoint, island.Id, props, elem))

			case IslandClientOnIdle:
				b.WriteString(fmt.Sprintf(`
onIdle().then(() => import('%s')).then(mod => mod.hydrate(%s, %s))
`, island.EntryPoint, props, elem))

			case IslandClientOnVisible:
				b.WriteString(fmt.Sprintf(`
onVisible(%s).then(() => import('%s')).then(mod => mod.hydrate(%s, %s))
`, elem, island.EntryPoint, props, elem))
			}
		}
		return b.String()
	},
}
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
