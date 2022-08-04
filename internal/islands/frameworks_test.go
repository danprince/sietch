package islands

import (
	"os"
	"path"
	"strings"
	"testing"
)

func compareStrings(t *testing.T, expect, actual string) {
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

func TestFrameworkClients(t *testing.T) {
	type props map[string]any

	islands := []*Island{
		{Id: "a", EntryPoint: "./Counter.tsx", Props: props{"count": 1}, Type: ClientOnLoad},
		{Id: "b", EntryPoint: "./Counter.tsx", Props: props{"count": 3}, Type: ClientOnIdle},
		{Id: "c", EntryPoint: "../Timer.tsx", Props: props{}, Type: ClientOnVisible},
	}

	tests := []Framework{Vanilla, Preact}
	cwd, _ := os.Getwd()

	for _, framework := range tests {
		t.Run(framework.Id+"-client", func(t *testing.T) {
			filename := path.Join(cwd, "testdata", framework.Id+"-client.js")
			data, err := os.ReadFile(filename)

			if err != nil {
				t.Fatal(err)
			}

			expect := string(data)
			actual := framework.clientEntryPoint(islands)
			compareStrings(t, expect, actual)
		})
	}

	for _, framework := range tests {
		t.Run(framework.Id+"-static", func(t *testing.T) {
			filename := path.Join(cwd, "testdata", framework.Id+"-static.js")
			data, err := os.ReadFile(filename)

			if err != nil {
				t.Fatal(err)
			}

			expect := string(data)
			actual := framework.staticEntryPoint(islands)
			compareStrings(t, expect, actual)
		})
	}
}
