package islands

import (
	"fmt"
	"regexp"
)

// Frameworks decide how to create the entry point files for bundling islands.
type Framework struct {
	Id              string
	extensions      []string
	jsxImportSource string
	clientEntry     func(filename string) string
	staticEntry     func(filename string) string
}

func (f *Framework) detect(filename string) bool {
	for _, ext := range f.extensions {
		if matched, _ := regexp.MatchString(ext, filename); matched {
			return true
		}
	}
	return false
}

var Vanilla = &Framework{
	Id: "vanilla",
	extensions: []string{
		`\.vanilla.(tsx?|jsx?)$`,
		`\.(ts|js)$`,
	},
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
	extensions: []string{
		`\.preact\.(tsx?|jsx?)$`,
		`\.(tsx|jsx)$`,
	},
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
