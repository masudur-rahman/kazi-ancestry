package person

import (
	"os"
	"sort"
	"testing"

	"github.com/masudur-rahman/kazi-ancestry/models"
	"github.com/masudur-rahman/kazi-ancestry/repos"

	"github.com/masudur-rahman/styx"
)

// fakeRepo is an in-memory PersonRepository. List mirrors the SQL repo's
// (position, id) ordering so service logic that depends on it (NormalizePositions,
// Create) is exercised faithfully.
type fakeRepo struct{ people []models.Person }

func (f *fakeRepo) WithUnitOfWork(styx.UnitOfWork) repos.PersonRepository { return f }
func (f *fakeRepo) List() ([]models.Person, error) {
	out := append([]models.Person(nil), f.people...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Position != out[j].Position {
			return out[i].Position < out[j].Position
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
func (f *fakeRepo) GetByID(id string) (*models.Person, error) {
	for i := range f.people {
		if f.people[i].ID == id {
			p := f.people[i]
			return &p, nil
		}
	}
	return nil, models.ErrPersonNotFound{ID: id}
}
func (f *fakeRepo) Add(p *models.Person) error { f.people = append(f.people, *p); return nil }
func (f *fakeRepo) Update(id string, p *models.Person) error {
	for i := range f.people {
		if f.people[i].ID == id {
			f.people[i] = *p
			return nil
		}
	}
	return models.ErrPersonNotFound{ID: id}
}
func (f *fakeRepo) SetPosition(id string, pos int) error {
	for i := range f.people {
		if f.people[i].ID == id {
			f.people[i].Position = pos
			return nil
		}
	}
	return models.ErrPersonNotFound{ID: id}
}
func (f *fakeRepo) Delete(id string) error {
	for i := range f.people {
		if f.people[i].ID == id {
			f.people = append(f.people[:i], f.people[i+1:]...)
			return nil
		}
	}
	return nil
}
func (f *fakeRepo) Count() (int, error) { return len(f.people), nil }
func (f *fakeRepo) DeleteAll() error    { f.people = nil; return nil }

func ptr(s string) *string { return &s }

// child positions under parentID, in List (position, id) order.
func positionsUnder(t *testing.T, svc *personService, parentID string) []int {
	t.Helper()
	people, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var pos []int
	for _, p := range people {
		if p.ParentID != nil && *p.ParentID == parentID {
			pos = append(pos, p.Position)
		}
	}
	return pos
}

// TestCreateAfterDelete: deleting a middle sibling then adding a new one must not
// reuse an existing position (max+1, not count).
func TestCreateAfterDelete(t *testing.T) {
	repo := &fakeRepo{people: []models.Person{
		{ID: "root", Name: "root"},
		{ID: "a", ParentID: ptr("root"), Name: "a", Position: 0},
		{ID: "b", ParentID: ptr("root"), Name: "b", Position: 1},
		{ID: "c", ParentID: ptr("root"), Name: "c", Position: 2},
	}}
	svc := NewPersonService(repo)

	if err := svc.Delete("b"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := svc.Create(&models.Person{ID: "d", ParentID: ptr("root"), Name: "d"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got := positionsUnder(t, svc, "root")
	// remaining a(0), c(2), new d must be 3 — no duplicate with c.
	seen := map[int]bool{}
	for _, p := range got {
		if seen[p] {
			t.Fatalf("duplicate position %d among %v", p, got)
		}
		seen[p] = true
	}
	if !seen[3] {
		t.Errorf("new child should be position 3 (max+1), got %v", got)
	}
}

// TestNormalizePositions: a group whose siblings all share position 0 is renumbered
// to a contiguous 0..n-1 (by the List (position, id) order); already-clean groups
// are left untouched and the call is idempotent.
func TestNormalizePositions(t *testing.T) {
	repo := &fakeRepo{people: []models.Person{
		{ID: "root", Name: "root"},
		{ID: "z", ParentID: ptr("root"), Name: "z", Position: 0},
		{ID: "x", ParentID: ptr("root"), Name: "x", Position: 0},
		{ID: "y", ParentID: ptr("root"), Name: "y", Position: 0},
	}}
	svc := NewPersonService(repo)

	if err := svc.NormalizePositions(); err != nil {
		t.Fatalf("NormalizePositions: %v", err)
	}
	// List order for the tied group is by id: x, y, z → positions 0,1,2.
	want := map[string]int{"x": 0, "y": 1, "z": 2}
	people, _ := svc.List()
	for _, p := range people {
		if p.ParentID != nil {
			if w, ok := want[p.ID]; ok && p.Position != w {
				t.Errorf("%s position = %d, want %d", p.ID, p.Position, w)
			}
		}
	}
	// idempotent: a second run changes nothing.
	before := positionsUnder(t, svc, "root")
	if err := svc.NormalizePositions(); err != nil {
		t.Fatalf("NormalizePositions (2nd): %v", err)
	}
	after := positionsUnder(t, svc, "root")
	if len(before) != len(after) {
		t.Fatalf("position count changed: %v -> %v", before, after)
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("not idempotent: %v -> %v", before, after)
		}
	}
}

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
