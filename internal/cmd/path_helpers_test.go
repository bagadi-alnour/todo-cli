package cmd

import "testing"

func TestNormalizePaths(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", nil, nil},
		{"single", []string{"src"}, []string{"src"}},
		{"commaSeparated", []string{"src/pkg,internal/api"}, []string{"src/pkg", "internal/api"}},
		{"mixedCommaAndRepeat", []string{"src/pkg,internal/api", "docs"}, []string{"src/pkg", "internal/api", "docs"}},
		{"trimsWhitespace", []string{" src ,  internal/api ,"}, []string{"src", "internal/api"}},
		{"dropsEmpty", []string{"", ",,"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePaths(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("len mismatch: got %v want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("at %d got %q want %q (full %v)", i, got[i], tt.want[i], got)
				}
			}
		})
	}
}
