// Package db wires the Postgres connection (via styx) and (re)seeds the family
// tree from the integer-id JSON source, assigning stable string slug ids.
package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	isql "github.com/masudur-rahman/styx/sql"
	"github.com/masudur-rahman/styx/sql/postgres"
	pglib "github.com/masudur-rahman/styx/sql/postgres/lib"

	"github.com/masudur-rahman/kazi-ancestry/internal/config"
	"github.com/masudur-rahman/kazi-ancestry/internal/models"
	"github.com/masudur-rahman/kazi-ancestry/internal/slug"
)

// tables managed by Sync, in dependency order.
var tables = []any{models.Person{}, models.User{}, models.Suggestion{}}

// Connect opens a Postgres connection and returns the styx engine.
func Connect(cfg config.Config) (isql.Engine, error) {
	conn, err := pglib.GetPostgresConnection(cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return postgres.NewPostgres(conn), nil
}

// Sync creates/updates the schema for all managed tables.
func Sync(ctx context.Context, db isql.Engine) error {
	if err := db.Sync(ctx, tables...); err != nil {
		return fmt.Errorf("sync schema: %w", err)
	}
	return nil
}

// rawPerson is the shape of the integer-id seed JSON (web/family.local.json).
type rawPerson struct {
	ID       int    `json:"id"`
	ParentID *int   `json:"parentId"`
	Name     string `json:"name"`
	Origin   string `json:"origin"`
	Alias    string `json:"alias"`
	Spouse   string `json:"spouse"`
	Birth    string `json:"birth"`
	Death    string `json:"death"`
	Note     string `json:"note"`
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
	// deterministic order: ascending original id
	sort.Slice(raw, func(i, j int) bool { return raw[i].ID < raw[j].ID })

	taken := map[string]bool{}
	idFor := make(map[int]string, len(raw))
	for _, r := range raw {
		parentName := ""
		if r.ParentID != nil {
			parentName = byID[*r.ParentID].Name
		}
		idFor[r.ID] = slug.Generate(r.Name, parentName, taken)
	}

	people := make([]models.Person, 0, len(raw))
	for _, r := range raw {
		var parent *string
		if r.ParentID != nil {
			pid := idFor[*r.ParentID]
			parent = &pid
		}
		p := models.Person{
			ID: idFor[r.ID], ParentID: parent, Name: r.Name,
			Origin: r.Origin, Alias: r.Alias, Spouse: r.Spouse,
			Birth: r.Birth, Death: r.Death, Note: r.Note,
		}
		p.SetTags(r.Tags)
		people = append(people, p)
	}
	return people, nil
}

// Count returns the number of rows in the person table.
func Count(ctx context.Context, db isql.Engine) (int, error) {
	var people []models.Person
	if err := db.Table("person").FindMany(ctx, &people); err != nil {
		return 0, err
	}
	return len(people), nil
}

// Seed imports the tree if the person table is empty. Idempotent.
func Seed(ctx context.Context, db isql.Engine, seedPath string) (int, error) {
	n, err := Count(ctx, db)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return n, nil // already seeded
	}
	return insertAll(ctx, db, seedPath)
}

// Reseed drops and recreates the person table, then imports fresh. Use this to
// regenerate ids after editing names. Leaves user/suggestion tables untouched.
func Reseed(ctx context.Context, db isql.Engine, seedPath string) (int, error) {
	if err := db.DropTable(ctx, "person"); err != nil {
		return 0, fmt.Errorf("drop person: %w", err)
	}
	if err := db.Sync(ctx, models.Person{}); err != nil {
		return 0, fmt.Errorf("resync person: %w", err)
	}
	return insertAll(ctx, db, seedPath)
}

func insertAll(ctx context.Context, db isql.Engine, seedPath string) (int, error) {
	people, err := BuildPeople(seedPath)
	if err != nil {
		return 0, err
	}
	for i := range people {
		if _, err := db.Table("person").InsertOne(ctx, &people[i]); err != nil {
			return 0, fmt.Errorf("insert %q: %w", people[i].ID, err)
		}
	}
	return len(people), nil
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
