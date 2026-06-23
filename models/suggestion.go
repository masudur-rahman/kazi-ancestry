package models

// Suggestion is a proposed edit awaiting admin review.
type Suggestion struct {
	ID          string `db:"id,pk" json:"id"`
	PersonID    string `db:"person_id,req" json:"personId"`
	Payload     string `db:"payload,req" json:"payload"` // JSON-encoded edit
	SubmittedBy string `db:"submitted_by,req" json:"submittedBy"`
	Status      string `db:"status,req" json:"status"` // pending|approved|rejected
	CreatedAt   int64  `db:"created_at" json:"createdAt"`
}

func (Suggestion) TableName() string { return "suggestion" }
