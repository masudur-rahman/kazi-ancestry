package services

import "github.com/masudur-rahman/kazi-ancestry/models"

// PersonService is the business-logic seam for the family tree.
type PersonService interface {
	List() ([]models.Person, error)
	Get(id string) (*models.Person, error)
	Create(person *models.Person) error
	Update(id string, person *models.Person) error
	Delete(id string) error
	Count() (int, error)
	// Seed imports the tree from seedPath if the table is empty (idempotent).
	Seed(seedPath string) (int, error)
	// Reseed clears the table and reimports, regenerating ids after name edits.
	Reseed(seedPath string) (int, error)
}
