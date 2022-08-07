package islands

import (
	"strings"
	"testing"
)

func TestFrameworkDetect(t *testing.T) {
	type test struct {
		framework *Framework
		filename  string
		detect    bool
	}

	tests := []test{
		{Preact, "explicit.preact.tsx", true},
		{Preact, "explicit.preact.jsx", true},
		{Preact, "implicit.tsx", true},
		{Preact, "implicit.jsx", true},
		{Preact, "plain.js", false},
		{Preact, "plain.ts", false},
		{Preact, "wrong.css", false},

		{Vanilla, "explicit.vanilla.ts", true},
		{Vanilla, "explicit.vanilla.js", true},
		{Vanilla, "implicit.ts", true},
		{Vanilla, "implicit.js", true},
		{Vanilla, "wrong.css", false},
	}

	for _, test := range tests {
		ok := test.framework.detect(test.filename)
		if ok != test.detect {
			t.Errorf("expected %s to detect %s as %v", test.framework.Id, test.filename, test.detect)
		}
	}
}

func TestFrameworkOutputs(t *testing.T) {
	type test struct {
		framework *Framework
		filename  string
		client    string
		static    string
	}

	tests := []test{
		{
			framework: Vanilla,
			filename:  "./counter.ts",
			client:    `export { hydrate } from "./counter.ts";`,
			static:    `export { render } from "./counter.ts";`,
		},
		{
			framework: Preact,
			filename:  "./counter.tsx",
			client: `
import { h, hydrate as _hydrate } from "preact";
import Component from "./counter.tsx";

export function hydrate(props, element) {
	return _hydrate(h(Component, props), element);
}
`,
			static: `import { h } from "preact";
import { render as _render } from "preact-render-to-string";
import Component from "./counter.tsx";

export function render(props, element) {
	return _render(h(Component, props));
}`,
		},
	}

	for _, test := range tests {
		t.Run(test.framework.Id, func(t *testing.T) {
			clientEntry := test.framework.clientEntry(test.filename)
			staticEntry := test.framework.staticEntry(test.filename)
			compareStrings(t, test.client, clientEntry)
			compareStrings(t, test.static, staticEntry)
		})
	}
}

func compareStrings(t *testing.T, expect, actual string) {
	t.Helper()
	expect = strings.Trim(expect, "\n ")
	actual = strings.Trim(actual, "\n ")

	if expect == actual {
		return
	}

	expectLines := strings.Split(expect, "\n")
	actualLines := strings.Split(actual, "\n")

	for index := range expectLines {
		actualLine := actualLines[index]
		expectLine := expectLines[index]
		if actualLine != expectLine {
			t.Fatalf(`line %d did not match
expect: %s
actual: %s`, index+1, expectLine, actualLine)
		}
	}
}
