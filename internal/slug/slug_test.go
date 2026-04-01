package slug

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"bliss2", "bliss2"},
		{"My Project", "my-project"},
		{"my-project", "my-project"},
		{"My  Weird  Name!", "my-weird-name"},
		{"Hello World", "hello-world"},
		{"  leading trailing  ", "leading-trailing"},
		{"under_score", "under-score"},
		{"already-slugified", "already-slugified"},
		{"Caps AND spaces", "caps-and-spaces"},
		{"foo/bar", "foo-bar"},
		{"foo.bar", "foo-bar"},
	}

	for _, tt := range tests {
		got := Slugify(tt.input)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
