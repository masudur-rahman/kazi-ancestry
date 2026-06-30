package person

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/masudur-rahman/kazi-ancestry/models"
	"github.com/masudur-rahman/kazi-ancestry/pkg/slug"
	"github.com/masudur-rahman/kazi-ancestry/repos"
)

type personService struct {
	repo repos.PersonRepository
}

func NewPersonService(repo repos.PersonRepository) *personService {
	return &personService{repo: repo}
}

func (s *personService) List() ([]models.Person, error)           { return s.repo.List() }
func (s *personService) Get(id string) (*models.Person, error)    { return s.repo.GetByID(id) }
func (s *personService) Update(id string, p *models.Person) error { return s.repo.Update(id, p) }

// Reorder rewrites the sibling order under parentID. It only accepts ids that
// are actually children of parentID (a guard against cross-parent shuffles),
// then writes each id's new 0-based position.
func (s *personService) Reorder(parentID string, orderedIDs []string) error {
	people, err := s.repo.List()
	if err != nil {
		return err
	}
	children := map[string]bool{}
	for _, p := range people {
		if p.ParentID != nil && *p.ParentID == parentID {
			children[p.ID] = true
		}
	}
	for _, id := range orderedIDs {
		if !children[id] {
			return fmt.Errorf("%q is not a child of %q", id, parentID)
		}
	}
	for i, id := range orderedIDs {
		if err := s.repo.SetPosition(id, i); err != nil {
			return err
		}
	}
	return nil
}

// Create adds a person, generating a stable slug id (shortest unique against
// existing ids) when one isn't supplied — the runtime path for new people.
func (s *personService) Create(p *models.Person) error {
	if p.Tags == "" {
		p.SetTags(nil)
	}
	people, err := s.repo.List()
	if err != nil {
		return err
	}
	taken := make(map[string]bool, len(people))
	var parentName string
	siblings := 0
	for _, e := range people {
		taken[e.ID] = true
		if p.ParentID != nil && e.ID == *p.ParentID {
			parentName = e.Name
		}
		if samePtr(e.ParentID, p.ParentID) {
			siblings++
		}
	}
	p.Position = siblings // append after existing siblings
	if p.ID == "" {
		p.ID = slug.Generate(p.Name, parentName, taken)
	}
	return s.repo.Add(p)
}
func (s *personService) Delete(id string) error { return s.repo.Delete(id) }
func (s *personService) Count() (int, error)    { return s.repo.Count() }

// samePtr reports whether two optional parent ids refer to the same parent
// (both root, or both the same id).
func samePtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// Seed imports the tree if the table is empty. Idempotent.
func (s *personService) Seed(seedPath string) (int, error) {
	n, err := s.repo.Count()
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return n, nil
	}
	return s.insertAll(seedPath)
}

// Reseed clears the table and reimports, regenerating ids after name edits.
func (s *personService) Reseed(seedPath string) (int, error) {
	if err := s.repo.DeleteAll(); err != nil {
		return 0, fmt.Errorf("clear people: %w", err)
	}
	return s.insertAll(seedPath)
}

func (s *personService) insertAll(seedPath string) (int, error) {
	people, err := BuildPeople(seedPath)
	if err != nil {
		return 0, err
	}
	for i := range people {
		if err := s.repo.Add(&people[i]); err != nil {
			return 0, fmt.Errorf("insert %q: %w", people[i].ID, err)
		}
	}
	return len(people), nil
}

// rawPerson is the shape of the integer-id seed JSON (web/family.local.json).
type rawPerson struct {
	ID       int      `json:"id"`
	ParentID *int     `json:"parentId"`
	Name     string   `json:"name"`
	Origin   string   `json:"origin"`
	Alias    string   `json:"alias"`
	Spouse   string   `json:"spouse"`
	Birth    string   `json:"birth"`
	Death    string   `json:"death"`
	Note     string   `json:"note"`
	Tags     []string `json:"tags"`
}

// BuildPeople reads the integer-id seed and converts it to slug-id Person rows,
// remapping every parentId to the parent's new string id. Pure (no DB) so it is
// unit-testable and reusable for "regenerate ids" runs.
func BuildPeople(seedPath string) ([]models.Person, error) {
	data, err := readSeed(seedPath)
	if err != nil {
		return nil, err
	}
	var raw []rawPerson
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse seed: %w", err)
	}

	byID := make(map[int]rawPerson, len(raw))
	for _, r := range raw {
		byID[r.ID] = r
	}
	sort.Slice(raw, func(i, j int) bool { return raw[i].ID < raw[j].ID }) // deterministic

	taken := map[string]bool{}
	idFor := make(map[int]string, len(raw))
	for _, r := range raw {
		parentName := ""
		if r.ParentID != nil {
			parentName = byID[*r.ParentID].Name
		}
		idFor[r.ID] = slug.Generate(r.Name, parentName, taken)
	}

	// per-parent sibling counter ("" = root) → 0-based Position in seed order,
	// giving the tree a stable, editable initial sibling order.
	pos := map[string]int{}
	people := make([]models.Person, 0, len(raw))
	for _, r := range raw {
		var parent *string
		key := ""
		if r.ParentID != nil {
			pid := idFor[*r.ParentID]
			parent = &pid
			key = pid
		}
		p := models.Person{
			ID: idFor[r.ID], ParentID: parent, Position: pos[key], Name: r.Name,
			Origin: r.Origin, Alias: r.Alias, Spouse: r.Spouse,
			Birth: r.Birth, Death: r.Death, Note: r.Note,
		}
		pos[key]++
		p.SetTags(r.Tags)
		people = append(people, p)
	}
	return people, nil
}

// readSeed loads seedPath, falling back to web/family.json (the committed sample)
// if the given path (default web/family.local.json) is absent.
func readSeed(seedPath string) ([]byte, error) {
	if data, err := os.ReadFile(seedPath); err == nil {
		return data, nil
	}
	const fallback = "web/family.json"
	data, err := os.ReadFile(fallback)
	if err != nil {
		return nil, fmt.Errorf("read seed (%s and %s): %w", seedPath, fallback, err)
	}
	return data, nil
}
