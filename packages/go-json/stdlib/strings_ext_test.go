package stdlib

import (
	"testing"
)

func TestStringsExt_SplitWords(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"hello_world", []string{"hello", "world"}},
		{"helloWorld", []string{"hello", "World"}},
		{"hello-world", []string{"hello", "world"}},
		{"Hello World", []string{"Hello", "World"}},
		{"XMLParser", []string{"XML", "Parser"}},
		{"already", []string{"already"}},
		{"", nil},
	}
	for _, tt := range tests {
		got := splitWords(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("splitWords(%q) = %v, want %v", tt.input, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestStringsExt_CamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "helloWorld"},
		{"hello-world", "helloWorld"},
		{"Hello World", "helloWorld"},
		{"already", "already"},
	}
	for _, tt := range tests {
		got := toCamelCase(tt.input, false)
		if got != tt.expected {
			t.Errorf("camelCase(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestStringsExt_PascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "HelloWorld"},
		{"hello-world", "HelloWorld"},
		{"already", "Already"},
	}
	for _, tt := range tests {
		got := toCamelCase(tt.input, true)
		if got != tt.expected {
			t.Errorf("pascalCase(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestStringsExt_SnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"helloWorld", "hello_world"},
		{"Hello World", "hello_world"},
		{"hello-world", "hello_world"},
		{"already", "already"},
	}
	for _, tt := range tests {
		got := toSnakeCase(tt.input)
		if got != tt.expected {
			t.Errorf("snakeCase(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestStringsExt_KebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"helloWorld", "hello-world"},
		{"Hello World", "hello-world"},
		{"hello_world", "hello-world"},
	}
	for _, tt := range tests {
		got := toKebabCase(tt.input)
		if got != tt.expected {
			t.Errorf("kebabCase(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestStringsExt_Slugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World!", "hello-world"},
		{"  Multiple   Spaces  ", "multiple-spaces"},
		{"Special @#$ Characters!", "special-characters"},
		{"already-slug", "already-slug"},
		{"CamelCase", "camelcase"},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.expected {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
