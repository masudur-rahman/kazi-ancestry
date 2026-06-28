package slug

import "testing"

func TestRomanize(t *testing.T) {
	cases := map[string]string{
		"তাহের আলী কাজী": "taher", // first token unique
		"মাসুদ":          "masud",
		"কণিকা":          "konika",
		"হাসান":          "hasan",
		"তানিয়া":        "taniya", // য়-glide
		"তৈয়ব আলী কাজী": "toiyob",
	}
	for name, want := range cases {
		taken := map[string]bool{}
		if got := Generate(name, "", taken); got != want {
			t.Errorf("Generate(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestShortestUnique(t *testing.T) {
	taken := map[string]bool{}
	// two শফিক under different parents -> second disambiguates by parent token
	if got := Generate("শফিক", "জিবুল", taken); got != "shofik" {
		t.Errorf("first শফিক = %q, want shofik", got)
	}
	if got := Generate("শফিক", "তৈয়ব", taken); got != "shofik-toiyob" {
		t.Errorf("second শফিক = %q, want shofik-toiyob", got)
	}
	// third শফিক, parent জিবুল still free as a token -> given-parent form
	if got := Generate("শফিক", "জিবুল", taken); got != "shofik-jibul" {
		t.Errorf("third শফিক = %q, want shofik-jibul", got)
	}
	// fourth, every candidate now taken -> numeric fallback
	if got := Generate("শফিক", "জিবুল", taken); got != "shofik-2" {
		t.Errorf("fourth শফিক = %q, want shofik-2", got)
	}
}
