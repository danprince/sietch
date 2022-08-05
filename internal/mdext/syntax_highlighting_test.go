package mdext

import (
	"reflect"
	"testing"

	_ "embed"
)

func TestParseHighlightRanges(t *testing.T) {
	type lines = []lineRange
	type result struct {
		lang   string
		ranges []lineRange
	}

	tests := map[string]result{
		"js":               {"js", lines{}},
		"js/":              {"js", lines{}},
		"js/0-1":           {"js", lines{}},
		"js/5-4":           {"js", lines{{4, 4}}},
		"js/5 - ":          {"js", lines{{4, 4}}},
		"js/5":             {"js", lines{{4, 4}}},
		"py/1-5":           {"py", lines{{0, 4}}},
		"rs/1,4":           {"rs", lines{{0, 0}, {3, 3}}},
		"tsx/1-2, 5, 9-20": {"tsx", lines{{0, 1}, {4, 4}, {8, 19}}},
		"tsx/1-2-3":        {"tsx", lines{{0, 1}}},
	}

	for input, expected := range tests {
		actualLang, actualRanges := parseHighlightRanges(input)

		if actualLang != expected.lang {
			t.Errorf(`expected language in "%s" to be "%s" but got "%s"`, input, expected.lang, actualLang)
		}

		if !reflect.DeepEqual(expected.ranges, actualRanges) {
			t.Errorf(`expected ranges in "%s" to be "%v" but got "%v"`, input, expected.ranges, actualRanges)
		}
	}
}
