package scraper

import "testing"

func TestShouldStop_EmptyPage(t *testing.T) {
	got := shouldStop(0, 50)
	want := true

	if got != want {
		t.Errorf("shouldStop(0, 50) = %v, want %v", got, want)
	}
}

func TestShouldStop_PartialPage(t *testing.T) {
	got := shouldStop(10, 50)
	want := true

	if got != want {
		t.Errorf("shouldStop(10, 50) = %v, want %v", got, want)
	}
}

func TestShouldStop_InvalidLimit(t *testing.T) {
	got := shouldStop(10, 0)
	want := true

	if got != want {
		t.Errorf("shouldStop(10, 0) = %v, want %v", got, want)
	}
}

func TestNextOffset_Range(t *testing.T) {
	table := []struct {
		name  string
		page  int
		limit int
		want  int
	}{
		{"first_page", 0, 50, 0},
		{"second_page", 1, 50, 50},
		{"third_page", 2, 50, 100},
		{"invalid_negative", -1, 50, 0},
		{"invalid_zero_limit", 1, 0, 0},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			got := nextOffset(tc.page, tc.limit)
			if got != tc.want {
				t.Errorf(
					"nextOffset(%v, %v) = %v, want %v",
					tc.page,
					tc.limit,
					got,
					tc.want,
				)
			}
		})
	}
}
