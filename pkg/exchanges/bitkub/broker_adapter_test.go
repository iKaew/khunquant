package bitkub

import "testing"

func TestNormalizeOrderHistoryLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "default", limit: 0, want: 20},
		{name: "negative", limit: -1, want: 20},
		{name: "requested", limit: 50, want: 50},
		{name: "max", limit: 100, want: 100},
		{name: "cap", limit: 200, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeOrderHistoryLimit(tt.limit); got != tt.want {
				t.Fatalf("normalizeOrderHistoryLimit(%d) = %d, want %d", tt.limit, got, tt.want)
			}
		})
	}
}
