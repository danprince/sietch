package islands

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Frameworks decide how to create the entry point files for bundling islands.
type Framework struct {
	Id               string
	importMap        map[string]string
	staticEntryPoint func(islands []*Island) string
	clientEntryPoint func(islands []*Island) string
}

var Vanilla = Framework{
	Id: "vanilla",
	staticEntryPoint: func(islands []*Island) string {
		var b strings.Builder
		for _, island := range islands {
			props, _ := json.Marshal(island.Props)

			b.WriteString(fmt.Sprintf(
				"import { render as $r%s } from '%s';\n",
				island.Id, island.EntryPoint,
			))

			b.WriteString(fmt.Sprintf(
				"$elements['%s'] = $r%s(%s);\n",
				island.Id, island.Id, props,
			))
		}
		return b.String()
	},
	clientEntryPoint: func(islands []*Island) string {
		var b strings.Builder

		b.WriteString("import { onIdle, onVisible } from '@sietch/client';\n\n")

		for _, island := range islands {
			id := island.Id
			src := island.EntryPoint
			elem := fmt.Sprintf(`document.getElementById('%s')`, id)
			props, _ := json.Marshal(island.Props)

			switch island.Type {
			case HydrateOnLoad:
				b.WriteString(fmt.Sprintf("import { hydrate as $h%s } from '%s';\n", id, src))
				b.WriteString(fmt.Sprintf("$h%s(%s, %s);\n", id, props, elem))
				b.WriteByte('\n')

			case HydrateOnIdle:
				b.WriteString( /*      */ "onIdle()\n")
				b.WriteString(fmt.Sprintf("  .then(() => import('%s'))\n", src))
				b.WriteString(fmt.Sprintf("  .then(md => md.hydrate(%s, %s));\n", props, elem))
				b.WriteByte('\n')

			case HydrateOnVisible:
				b.WriteString(fmt.Sprintf("let $e%s = %s;\n", id, elem))
				b.WriteString(fmt.Sprintf("onVisible($e%s)\n", id))
				b.WriteString(fmt.Sprintf("  .then(() => import('%s'))\n", src))
				b.WriteString(fmt.Sprintf("  .then(md => md.hydrate(%s, $e%s));", props, id))
				b.WriteByte('\n')
			}
		}
		return b.String()
	},
}

var Preact = Framework{
	Id: "preact",
	staticEntryPoint: func(islands []*Island) string {
		var b strings.Builder

		b.WriteString("import { h } from 'preact';\n")
		b.WriteString("import { renderToString as render } from 'preact-render-to-string';\n")

		for _, island := range islands {
			props := island.Props

			b.WriteString(fmt.Sprintf(
				"import $c%s from '%s';\n",
				island.Id, island.EntryPoint,
			))

			b.WriteString(fmt.Sprintf(
				"$elements['%s'] = render(h($c%s, %s));\n",
				island.Id, island.Id, props,
			))
		}
		return b.String()
	},
	clientEntryPoint: func(islands []*Island) string {
		var b strings.Builder

		b.WriteString("import { onIdle, onVisible } from '@sietch/client';\n")
		b.WriteString("import { h, hydrate } from 'preact';\n\n")

		for _, island := range islands {
			id := island.Id
			src := island.EntryPoint
			elem := fmt.Sprintf(`document.getElementById('%s')`, island.Id)
			props := island.Props

			switch island.Type {
			case HydrateOnLoad:
				b.WriteString(fmt.Sprintf("import $c%s from '%s';\n", id, src))
				b.WriteString(fmt.Sprintf("hydrate(h($c%s, %s), %s);\n", id, props, elem))
				b.WriteByte('\n')

			case HydrateOnIdle:
				b.WriteString( /*      */ "onIdle()\n")
				b.WriteString(fmt.Sprintf("  .then(() => import('%s'))\n", src))
				b.WriteString(fmt.Sprintf("  .then(md => hydrate(h(md.default, %s), %s));\n", props, elem))
				b.WriteByte('\n')

			case HydrateOnVisible:
				b.WriteString(fmt.Sprintf("let $e%s = %s;\n", id, elem))
				b.WriteString(fmt.Sprintf("onVisible($e%s)\n", id))
				b.WriteString(fmt.Sprintf("  .then(() => import('%s'))\n", src))
				b.WriteString(fmt.Sprintf("  .then(md => hydrate(h(md.default, %s), $e%s));\n", props, id))
				b.WriteByte('\n')
			}
		}
		return b.String()
	},
}

var PreactRemote = Framework{
	Id: "preact-remote",
	importMap: map[string]string{
		"preact":                  "https://esm.sh/preact@10.10.0",
		"preact/hooks":            "https://esm.sh/preact@10.10.0/hooks",
		"preact/jsx-runtime":      "https://esm.sh/preact@10.10.0/jsx-runtime",
		"preact-render-to-string": "https://esm.sh/preact@10.10.0/compat/server",
	},
	staticEntryPoint: Preact.staticEntryPoint,
	clientEntryPoint: Preact.clientEntryPoint,
}
