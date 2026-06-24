package web

import (
	"testing"

	"github.com/masudur-rahman/kazi-ancestry/models"
)

func TestNamesOnly(t *testing.T) {
	root := "root"
	in := []models.Person{
		{ID: "root", Name: "Root", Origin: "Town", Alias: "R", Note: "secret", Tags: `["died_young"]`},
		{ID: "kid", ParentID: &root, Name: "Kid", Spouse: "S", Birth: "1990"},
	}
	out := namesOnly(in)

	if len(out) != len(in) {
		t.Fatalf("len = %d, want %d", len(out), len(in))
	}
	// tree shape + name kept
	if out[0].ID != "root" || out[0].Name != "Root" {
		t.Errorf("root id/name not preserved: %+v", out[0])
	}
	if out[1].ParentID == nil || *out[1].ParentID != "root" {
		t.Errorf("parent link not preserved: %+v", out[1])
	}
	// everything else blanked
	if out[0].Origin != "" || out[0].Alias != "" || out[0].Note != "" || out[1].Spouse != "" || out[1].Birth != "" {
		t.Errorf("non-name fields leaked: %+v %+v", out[0], out[1])
	}
	if got := out[0].TagList(); len(got) != 0 {
		t.Errorf("tags leaked: %v", got)
	}
}
