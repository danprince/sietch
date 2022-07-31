package islands

import (
	"encoding/json"
	"fmt"
	"strings"
)

var React = Framework{
	importMap:       map[string]string{},
	jsxImportSource: "react/jsx-runtime",
	createRenderScript: func(ctx *Ctx) (string, error) {
		var script strings.Builder

		if ctx.needsSSR() {
			script.WriteString("import React from 'react'\n")
			script.WriteString("import { renderToString as render } from 'react-dom/server';\n")
			script.WriteString("globalThis.$ = {};")
		}

		for id, el := range ctx.Elements {
			if el.SSR {
				props, err := json.Marshal(el.Props)

				if err != nil {
					return "", err
				}

				imports := fmt.Sprintf("import %s from '%s';", id, el.entryPoint)
				renders := fmt.Sprintf("$['%s'] = render(React.createElement(%s, %s));", el.marker, id, string(props))
				script.WriteString(imports)
				script.WriteString(renders)
			}
		}

		return script.String(), nil
	},
	createHydrateScript: func(ctx *Ctx) (string, error) {
		var script strings.Builder

		if ctx.needsCSR() {
			script.WriteString("import React from 'react'\n")
			script.WriteString("import { hydrate } from 'react-dom';\n")
		}

		for id, el := range ctx.Elements {
			if el.CSR {
				props, err := json.Marshal(el.Props)

				if err != nil {
					return "", err
				}

				element := fmt.Sprintf("document.getElementById('%s')", id)
				imports := fmt.Sprintf("import %s from '%s';\n", id, el.entryPoint)
				renders := fmt.Sprintf("hydrate(React.createElement(%s, %s), %s);\n", id, string(props), element)
				script.WriteString(imports)
				script.WriteString(renders)
			} else if el.SSR {
				// Import SSR elements (but don't render) just in case they require CSS files
				script.WriteString(fmt.Sprintf("import '%s';\n", el.entryPoint))
			}
		}

		return script.String(), nil
	},
}

var Preact = Framework{
	importMap:       map[string]string{},
	jsxImportSource: "preact/jsx-runtime",
	createRenderScript: func(ctx *Ctx) (string, error) {
		var script strings.Builder

		if ctx.needsSSR() {
			script.WriteString("import { h } from 'preact'\n")
			script.WriteString("import { render } from 'preact-render-to-string';\n")
			script.WriteString("globalThis.$ = {};")
		}

		for id, el := range ctx.Elements {
			if el.SSR {
				props, err := json.Marshal(el.Props)

				if err != nil {
					return "", err
				}

				imports := fmt.Sprintf("import %s from '%s';", id, el.entryPoint)
				renders := fmt.Sprintf("$['%s'] = render(h(%s, %s));", el.marker, id, string(props))
				script.WriteString(imports)
				script.WriteString(renders)
			}
		}

		return script.String(), nil
	},
	createHydrateScript: func(ctx *Ctx) (string, error) {
		var script strings.Builder

		if ctx.needsCSR() {
			script.WriteString("import { h, hydrate } from 'preact'\n")
		}

		for id, el := range ctx.Elements {
			if el.CSR {
				props, err := json.Marshal(el.Props)

				if err != nil {
					return "", err
				}

				element := fmt.Sprintf("document.getElementById('%s')", id)
				imports := fmt.Sprintf("import %s from '%s';\n", id, el.entryPoint)
				renders := fmt.Sprintf("hydrate(h(%s, %s), %s);\n", id, string(props), element)
				script.WriteString(imports)
				script.WriteString(renders)
			} else if el.SSR {
				// Import SSR elements (but don't render) just in case they require CSS files
				script.WriteString(fmt.Sprintf("import '%s';\n", el.entryPoint))
			}
		}

		return script.String(), nil
	},
}

var PreactCDN = Framework{
	importMap: map[string]string{
		"preact":                  "https://esm.sh/preact@10.7.2",
		"preact/hooks":            "https://esm.sh/preact@10.7.2/hooks",
		"preact-render-to-string": "https://esm.sh/preact-render-to-string@5.2.0?external=preact",
		"preact/jsx-runtime":      "https://esm.sh/preact@10.7.2/jsx-runtime",
	},
	jsxImportSource:     "https://esm.sh/preact@10.7.2",
	createRenderScript:  Preact.createRenderScript,
	createHydrateScript: Preact.createHydrateScript,
}

var ReactCDN = Framework{
	importMap: map[string]string{
		"react":            "https://esm.sh/react@17.0.2",
		"react-dom":        "https://esm.sh/react-dom@17.0.2",
		"react-dom/server": "https://esm.sh/react-dom@17.0.2/server",
	},
	jsxImportSource:     "https://esm.sh/react-dom@17.0.2",
	createRenderScript:  React.createRenderScript,
	createHydrateScript: React.createHydrateScript,
}
