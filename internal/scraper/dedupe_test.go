package scraper

import (
	"reflect"
	"testing"
)

func TestMergeUnique_AddSingle_ReturnsSingle(t *testing.T) {
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

func TestMergeUnique_PopulatedExisting_AddListOfUniques_ReturnsCombined(t *testing.T) {
	existing := []Company{
		{
			Name:             "amazon",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "facebook",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "apple",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "nvidia",
			ShortDescription: "description",
			Size:             "100",
		},
	}
	page := []Company{
		{
			Name:             "tesla",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "disney",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "cisco",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "ford",
			ShortDescription: "description",
			Size:             "100",
		},
	}

	seen := map[string]struct{}{}
	got := mergeUnique(existing, page, seen)
	want := []Company{
		{
			Name:             "amazon",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "facebook",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "apple",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "nvidia",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "tesla",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "disney",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "cisco",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "ford",
			ShortDescription: "description",
			Size:             "100",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMergeUnique_AddSame_ReturnsNoChange(t *testing.T) {
	existing := []Company{
		{
			Name:             "acme",
			ShortDescription: "description",
			Size:             "100",
		},
	}
	page := []Company{
		{
			Name:             "acme",
			ShortDescription: "description",
			Size:             "100",
		},
	}

	seen := makeListOfSeenNames([]string{"acme"})
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

func TestMergeUnique_PopulatedExisting_AddSameList_ReturnsNoChange(t *testing.T) {
	existing := []Company{
		{
			Name:             "amazon",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "facebook",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "apple",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "nvidia",
			ShortDescription: "description",
			Size:             "100",
		},
	}
	page := []Company{
		{
			Name:             "amazon",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "facebook",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "apple",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "nvidia",
			ShortDescription: "description",
			Size:             "100",
		},
	}

	seen := makeListOfSeenNames([]string{"amazon", "facebook", "apple", "nvidia"})
	got := mergeUnique(existing, page, seen)
	uniqueTest(t, got)
}

func TestMergeUnique_AllRandomCaseSameNameCompanies_ReturnsNoChange(t *testing.T) {
	existing := []Company{} // Should not check existing

	page := []Company{
		{
			Name:             "AMazon",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "FaceBook",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "APPle",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "nVIdiA",
			ShortDescription: "description",
			Size:             "100",
		},
	}

	seen := makeListOfSeenNames([]string{"amazon", "facebook", "apple", "nvidia"})
	got := mergeUnique(existing, page, seen)
	want := existing

	testEquality(t, got, want)
}

func TestMergeUnique_RandomMixedCompanies_ReturnsCorrectList(t *testing.T) {
	existing := []Company{}

	page := []Company{
		{
			Name:             "AMazon",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "Weyland Yutani",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "ceraVe",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "FaceBook",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "APPle",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "squareSPACE",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "nVIdiA",
			ShortDescription: "description",
			Size:             "100",
		},
	}

	seen := makeListOfSeenNames([]string{"amazon", "facebook", "apple", "nvidia"})
	got := mergeUnique(existing, page, seen)
	want := []Company{
		{
			Name:             "weyland yutani",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "cerave",
			ShortDescription: "description",
			Size:             "100",
		},
		{
			Name:             "squarespace",
			ShortDescription: "description",
			Size:             "100",
		},
	}
	testEquality(t, got, want)
}

func testEquality(t *testing.T, got, want []Company) {
	if !reflect.DeepEqual(got, want) {
		if len(got) != len(want) {
			t.Errorf("len(got)=%d len(want)=%d", len(got), len(want))
		}
		for i, c := range got {
			t.Errorf("got[%d] = %+v", i, c)
		}
		t.Errorf("\n")
		for i, c := range want {
			t.Errorf("want[%d] = %+v", i, c)
		}
	}
}

func uniqueTest(t *testing.T, got []Company) {
	counts := map[string]int{}
	for _, c := range got {
		counts[c.Name]++ // increment the count of appearance of a name in the got list
	}

	for name, count := range counts {
		if count > 1 {
			t.Fatalf("duplicate company in result: name=%q count=%d", name, count)
		}
	}
}

func makeListOfSeenNames(names []string) map[string]struct{} {
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		seen[n] = struct{}{}
	}
	return seen
}
