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

var Preact = Framework{
	staticEntryPoint: func(islands []*Island) string {
		var b strings.Builder
		b.WriteString(`
import { h } from 'preact';
import { renderToString as render } from 'preact-render-to-string';
		`)
		for _, island := range islands {
			props, _ := json.Marshal(island.Props)
			b.WriteString(fmt.Sprintf("import { default as C%s } from '%s';\n", island.Id, island.EntryPoint))
			b.WriteString(fmt.Sprintf("$elements['%s'] = render(h(C%s, %s));\n", island.Id, island.Id, props))
		}
		return b.String()
	},
	clientEntryPoint: func(islands []*Island) string {
		var b strings.Builder
		b.WriteString(`
import { onIdle, onVisible } from "@sietch/client";
import { hydrate, h } from "preact";
`)

		for _, island := range islands {
			elem := fmt.Sprintf("document.getElementById('%s')", island.Id)
			props, _ := json.Marshal(island.Props)

			switch island.Type {
			case IslandClientOnLoad:
				b.WriteString(fmt.Sprintf(`
import { default as C%s } from '%s';
hydrate(h(C%s, %s), %s);
`, island.Id, island.EntryPoint, island.Id, props, elem))

			case IslandClientOnIdle:
				b.WriteString(fmt.Sprintf(`
onIdle().then(() => import('%s')).then(mod => hydrate(h(mod.default, %s), %s));
`, island.EntryPoint, props, elem))

			case IslandClientOnVisible:
				b.WriteString(fmt.Sprintf(`
onVisible(%s).then(() => import('%s')).then(mod => hydrate(h(mod.default, %s), %s));
`, elem, island.EntryPoint, props, elem))
			}
		}
		return b.String()
	},
}

var PreactRemote = Framework{
	importMap: map[string]string{
		"preact":                  "https://esm.sh/preact@10.10.0",
		"preact/hooks":            "https://esm.sh/preact@10.10.0/hooks",
		"preact/jsx-runtime":      "https://esm.sh/preact@10.10.0/jsx-runtime",
		"preact-render-to-string": "https://esm.sh/preact@10.10.0/compat/server",
	},
	staticEntryPoint: Preact.staticEntryPoint,
	clientEntryPoint: Preact.clientEntryPoint,
}
