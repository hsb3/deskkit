package tui

import "testing"

// TestShouldAutoFollow: streamed content auto-follows to the bottom only when the viewport was
// already pinned there (scroll percent at or past 1.0); a reader scrolled up keeps their position.
// The viewport reports 1.0 whenever the content fits, so a short/fresh transcript still follows.
func TestShouldAutoFollow(t *testing.T) {
	cases := []struct {
		name    string
		percent float64
		want    bool
	}{
		{"at bottom follows", 1.0, true},
		{"past bottom follows", 1.5, true},
		{"content fits reports one and follows", 1.0, true},
		{"scrolled up does not follow", 0.5, false},
		{"at top does not follow", 0.0, false},
		{"just below bottom does not follow", 0.999, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldAutoFollow(tc.percent); got != tc.want {
				t.Errorf("shouldAutoFollow(%v) = %v, want %v", tc.percent, got, tc.want)
			}
		})
	}
}
