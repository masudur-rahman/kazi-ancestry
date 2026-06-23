package models

// User is an authenticated, allowlisted member. Role: viewer|contributor|admin.
type User struct {
	ID        string `db:"id,pk" json:"id"`
	Email     string `db:"email,uq req" json:"email"`
	Name      string `db:"name,req" json:"name"`
	Role      string `db:"role,req" json:"role"`
	CreatedAt int64  `db:"created_at" json:"createdAt"`
}

func (User) TableName() string { return "user_account" }
