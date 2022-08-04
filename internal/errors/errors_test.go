package errors

import "testing"

func TestLoc(t *testing.T) {
	type test struct {
		input  string
		offset int
		line   int
		col    int
	}

	tests := []test{
		{
			input:  "hello\nworld",
			offset: 0,
			line:   1,
			col:    1,
		},
		{
			input:  "hello\nworld",
			offset: 6,
			line:   2,
			col:    1,
		},
		{
			input:  "hello\nworld",
			offset: 8,
			line:   2,
			col:    3,
		},
	}

	for _, tc := range tests {
		line, col := loc(tc.input, tc.offset)
		if line != tc.line || col != tc.col {
			t.Errorf("expected %d:%d, got %d:%d", tc.line, tc.col, line, col)
		}
	}
}
