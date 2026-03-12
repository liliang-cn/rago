package search

import "testing"

func TestExpandKeywords(t *testing.T) {
	terms := ExpandKeywords("write_file create golang file")
	want := map[string]struct{}{
		"write":      {},
		"file":       {},
		"create":     {},
		"golang":     {},
		"write_file": {},
	}
	for term := range want {
		found := false
		for _, actual := range terms {
			if actual == term {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected expanded keyword %q in %v", term, terms)
		}
	}
}

func TestRank(t *testing.T) {
	results := Rank("write golang file", []Document{
		{ID: "list_directory", Text: "list files in workspace"},
		{ID: "write_file", Text: "write a file into workspace with content"},
		{ID: "search_web", Text: "search the web"},
	}, 3, nil)

	if len(results) == 0 {
		t.Fatal("expected ranked results")
	}
	if results[0].ID != "write_file" {
		t.Fatalf("expected write_file to rank first, got %#v", results)
	}
}
