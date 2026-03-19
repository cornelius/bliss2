package todo

import (
	"testing"
)

func TestParse_titleOnly(t *testing.T) {
	content := "Feed the penguins\n"
	got, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Title != "Feed the penguins" {
		t.Errorf("Title = %q, want %q", got.Title, "Feed the penguins")
	}
	if got.Body != "" {
		t.Errorf("Body = %q, want empty", got.Body)
	}
}

func TestParse_withBody(t *testing.T) {
	content := "Feed the penguins\n\nMake sure to bring the fish.\nCheck with the zookeeper.\n"
	got, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Title != "Feed the penguins" {
		t.Errorf("Title = %q, want %q", got.Title, "Feed the penguins")
	}
	wantBody := "Make sure to bring the fish.\nCheck with the zookeeper."
	if got.Body != wantBody {
		t.Errorf("Body = %q, want %q", got.Body, wantBody)
	}
}

func TestFormat_roundtrip(t *testing.T) {
	tests := []struct {
		name string
		todo Todo
	}{
		{
			name: "title only",
			todo: Todo{Title: "Buy groceries"},
		},
		{
			name: "with body",
			todo: Todo{
				Title: "Call the vet",
				Body:  "Ask about the appointment.\nBring vaccination records.",
			},
		},
		{
			name: "apostrophe in title",
			todo: Todo{Title: "Fix John's bug"},
		},
		{
			name: "double quotes in title",
			todo: Todo{Title: `He said "hello"`},
		},
		{
			name: "mixed special chars",
			todo: Todo{Title: `It's a "test" & more`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := Format(tt.todo)
			parsed, err := Parse(formatted)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if parsed.Title != tt.todo.Title {
				t.Errorf("Title = %q, want %q", parsed.Title, tt.todo.Title)
			}
			if parsed.Body != tt.todo.Body {
				t.Errorf("Body = %q, want %q", parsed.Body, tt.todo.Body)
			}
		})
	}
}
