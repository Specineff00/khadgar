package scraper

import (
	"reflect"
	"testing"
)

func TestMergeUnique_AddSingle(t *testing.T) {
	existing := []Company{}
	page := []Company{
		{
			Name:             "acme",
			ShortDescription: "description",
			Size:             "100",
		},
	}

	seen := map[string]struct{}{}
	got := mergeUnique(existing, page, seen)
	want := []Company{
		{
			Name:             "acme",
			ShortDescription: "description",
			Size:             "100",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
