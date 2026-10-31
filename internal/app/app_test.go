package app

import "testing"

// TestAppSkeleton is the Phase 1 walking-skeleton test: a trivial
// table-driven test giving go test -race a real target in this package (D-01).
func TestAppSkeleton(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "package app builds and race-tests",
			got:  "app",
			want: "app",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}
