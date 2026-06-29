package person

import (
	"os"
	"testing"
)

// TestBuildPeople verifies slug assignment + parentId remap over the seed source.
// It prefers the real tree (web/family.local.json) when present, else the committed
// fictional sample (web/family.json) so the test is deterministic in CI, where the
// real data is gitignored. Exercises the transliterator and the shortest-unique
// collision ladder end to end — no DB.
func TestBuildPeople(t *testing.T) {
	seed := "../../web/family.local.json"
	if _, err := os.Stat(seed); err != nil {
		seed = "../../web/family.json" // committed sample (real data is gitignored)
	}
	people, err := BuildPeople(seed)
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

	// Each parent's children must carry contiguous 0-based positions in seed order.
	byParent := map[string][]int{}
	for _, p := range people {
		key := ""
		if p.ParentID != nil {
			key = *p.ParentID
		}
		byParent[key] = append(byParent[key], p.Position)
	}
	for parent, positions := range byParent {
		for i, pos := range positions {
			if pos != i {
				t.Errorf("parent %q child #%d has position %d, want %d", parent, i, pos, i)
			}
		}
	}
}
