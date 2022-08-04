package builder

import "testing"

func TestShortHash(t *testing.T) {
	tests := map[string]string{
		"hello":   "4f9f",
		"goodbye": "46d7",
	}
	for input, expected := range tests {
		actual := shortHash(input)
		if actual != expected {
			t.Errorf("expected %s to hash as %s but got %s", input, expected, actual)
		}
	}
}

func TestLessAny(t *testing.T) {
	tests := map[[2]any]bool{
		{1, 2}:     true,
		{"a", "b"}: true,
		{"b", "a"}: false,
		{2, 1}:     false,

		// Mixed types are always false
		{2, false}: false,
		{false, 2}: false,
	}
	for input, expected := range tests {
		actual := lessAny(input[0], input[1])
		if actual != expected {
			if expected {
				t.Errorf("expected %v to be less than %v", input[0], input[1])
			} else {
				t.Errorf("expected %v to be less than %v", input[1], input[0])
			}
		}
	}
}
