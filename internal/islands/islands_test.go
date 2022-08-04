package islands

import "testing"

func TestIslandString(t *testing.T) {
	type test struct {
		island Island
		output string
	}

	tests := []test{
		{
			island: Island{Id: "A", Type: Static},
			output: `<!-- A -->`,
		},
		{
			island: Island{Id: "A", Type: HydrateOnIdle, ClientOnly: true},
			output: `<div id="A"></div>`,
		},
		{
			island: Island{Id: "A", Type: HydrateOnIdle},
			output: `<div id="A"><!-- A --></div>`,
		},
	}

	for _, tc := range tests {
		expect := tc.output
		actual := tc.island.String()
		if expect != actual {
			t.Errorf(`island strings did not match
island: %+v
expect: %s
actual: %s`, tc.island, expect, actual)
		}
	}
}
