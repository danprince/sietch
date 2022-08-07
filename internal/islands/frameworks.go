package islands

import (
	"fmt"
	"regexp"
)

// Frameworks decide how to create the entry point files for bundling islands.
type Framework struct {
	Id              string
	implicitExt     string
	explicitExt     string
	jsxImportSource string
	clientEntry     func(filename string) string
	staticEntry     func(filename string) string
}

var Vanilla = &Framework{
	Id:          "vanilla",
	explicitExt: `\.vanilla.(tsx?|jsx?)$`,
	implicitExt: `\.(ts|js)$`,
	staticEntry: func(filename string) string {
		return fmt.Sprintf(`export { render } from "%s";`, filename)
	},
	clientEntry: func(filename string) string {
		return fmt.Sprintf(`export { hydrate } from "%s";`, filename)
	},
}

var Preact = &Framework{
	Id:              "preact",
	jsxImportSource: "preact",
	explicitExt:     `\.preact.(tsx?|jsx?)$`,
	implicitExt:     `\.(tsx|jsx)$`,
	staticEntry: func(filename string) string {
		return fmt.Sprintf(`
import { h } from "preact";
import { render as _render } from "preact-render-to-string";
import Component from "%s";

export function render(props, element) {
	return _render(h(Component, props));
}`, filename)
	},
	clientEntry: func(filename string) string {
		return fmt.Sprintf(`
import { h, hydrate as _hydrate } from "preact";
import Component from "%s";

export function hydrate(props, element) {
	return _hydrate(h(Component, props), element);
}`, filename)
	},
}

func detectFramework(frameworks []*Framework, importUrl string) (*Framework, error) {
	for _, f := range frameworks {
		ok, _ := regexp.MatchString(f.explicitExt, importUrl)
		if ok {
			return f, nil
		}
	}

	for _, f := range frameworks {
		if ok, _ := regexp.MatchString(f.implicitExt, importUrl); ok {
			return f, nil
		}
	}

	return nil, fmt.Errorf("no islands framework found for: %s", importUrl)
}
