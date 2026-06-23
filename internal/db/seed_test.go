package db

import "testing"

// TestBuildPeople verifies slug assignment + parentId remap over the seed source
// (web/family.local.json, or the committed sample if absent). It exercises the
// transliterator and the shortest-unique collision ladder end to end — no DB.
func TestBuildPeople(t *testing.T) {
	people, err := BuildPeople("../../web/family.local.json")
	if err != nil {
		t.Fatalf("BuildPeople: %v", err)
	}
	if len(people) == 0 {
		t.Fatal("no people built")
	}

	ids := make(map[string]bool, len(people))
	for _, p := range people {
		if p.ID == "" {
			t.Errorf("empty id for %q", p.Name)
		}
		if ids[p.ID] {
			t.Errorf("duplicate id %q", p.ID)
		}
		ids[p.ID] = true
	}

	var roots int
	for _, p := range people {
		if p.ParentID == nil {
			roots++
			continue
		}
		if !ids[*p.ParentID] {
			t.Errorf("%q has dangling parent %q", p.ID, *p.ParentID)
		}
	}
	if roots != 1 {
		t.Errorf("want exactly 1 root, got %d", roots)
	}
}
