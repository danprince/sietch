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
		"js/5-4":           {"js", lines{{5, 5}}},
		"js/5 - ":          {"js", lines{{5, 5}}},
		"js/5":             {"js", lines{{5, 5}}},
		"py/1-5":           {"py", lines{{1, 5}}},
		"rs/1,4":           {"rs", lines{{1, 1}, {4, 4}}},
		"tsx/1-2, 5, 9-20": {"tsx", lines{{1, 2}, {5, 5}, {9, 20}}},
		"tsx/1-2-3":        {"tsx", lines{{1, 2}}},
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
