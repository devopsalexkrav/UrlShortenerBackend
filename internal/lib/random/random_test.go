package random

import (
	"strings"
	"testing"
)

func TestNewRandomString_Length(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"empty", 0},
		{"one", 1},
		{"short", 6},
		{"medium", 10},
		{"long", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewRandomString(tt.size)
			if len(result) != tt.size {
				t.Errorf("NewRandomString(%d) returned string of length %d, want %d", tt.size, len(result), tt.size)
			}
		})
	}
}

func TestNewRandomString_ValidChars(t *testing.T) {
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	result := NewRandomString(1000)

	for i, c := range result {
		if !strings.ContainsRune(validChars, c) {
			t.Errorf("NewRandomString() contains invalid character %q at position %d", c, i)
		}
	}
}

func TestNewRandomString_Uniqueness(t *testing.T) {
	const iterations = 100
	const size = 10

	generated := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		s := NewRandomString(size)
		if generated[s] {
			t.Errorf("NewRandomString() generated duplicate string: %s", s)
		}
		generated[s] = true
	}
}

func TestNewRandomString_Randomness(t *testing.T) {
	s1 := NewRandomString(20)
	s2 := NewRandomString(20)

	if s1 == s2 {
		t.Errorf("NewRandomString() generated identical strings: %s and %s", s1, s2)
	}
}

func BenchmarkNewRandomString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewRandomString(10)
	}
}

