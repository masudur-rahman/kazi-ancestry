package models

import (
	"encoding/json"
	"time"
)

// Person is a node in the family tree. Tags is stored as a JSON-encoded string
// column (styx maps it as TEXT); the API/seed shape exposes it as a string array.
type Person struct {
	// req on the text fields makes styx always write them (even ""), so columns
	// never hold SQL NULL and scan cleanly back into Go strings.
	ID       string  `db:"id,pk" json:"id"`
	ParentID *string `db:"parent_id" json:"parentId"` // nil = root (stored NULL)
	Name     string  `db:"name,req" json:"name"`
	Origin   string  `db:"origin,req" json:"origin"`
	Alias    string  `db:"alias,req" json:"alias"`
	Spouse   string  `db:"spouse,req" json:"spouse"`
	Birth    string  `db:"birth,req" json:"birth"`
	Death    string  `db:"death,req" json:"death"`
	Note     string  `db:"note,req" json:"note"`
	Tags     string  `db:"tags,req" json:"-"` // JSON-encoded []string, e.g. ["died_young"]
}

// TagList decodes the stored Tags JSON into a slice (nil/invalid -> empty).
func (p Person) TagList() []string {
	if p.Tags == "" {
		return []string{}
	}
	var t []string
	if err := json.Unmarshal([]byte(p.Tags), &t); err != nil || t == nil {
		return []string{}
	}
	return t
}

// SetTags JSON-encodes a slice into the Tags column.
func (p *Person) SetTags(tags []string) {
	if len(tags) == 0 {
		p.Tags = "[]"
		return
	}
	b, _ := json.Marshal(tags)
	p.Tags = string(b)
}

// MarshalJSON emits the flat shape app.js consumes (tags as an array), so the
// injected seed is byte-compatible with the legacy web/family.json.
func (p Person) MarshalJSON() ([]byte, error) {
	type alias Person // avoid recursion
	return json.Marshal(struct {
		alias
		Tags []string `json:"tags"`
	}{alias(p), p.TagList()})
}

// User is an authenticated, allowlisted member. Role: viewer|contributor|admin.
type User struct {
	ID        string    `db:"id,pk" json:"id"`
	Email     string    `db:"email,uq req" json:"email"`
	Name      string    `db:"name" json:"name"`
	Role      string    `db:"role,req" json:"role"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}

// Suggestion is a proposed edit awaiting admin review.
type Suggestion struct {
	ID          string    `db:"id,pk" json:"id"`
	PersonID    string    `db:"person_id" json:"personId"`
	Payload     string    `db:"payload" json:"payload"` // JSON-encoded edit
	SubmittedBy string    `db:"submitted_by" json:"submittedBy"`
	Status      string    `db:"status,req" json:"status"` // pending|approved|rejected
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}
