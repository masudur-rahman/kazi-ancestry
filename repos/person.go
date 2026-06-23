package repos

import (
	"github.com/masudur-rahman/kazi-ancestry/models"

	"github.com/masudur-rahman/styx"
)

// PersonRepository is the data-access seam for the family tree.
type PersonRepository interface {
	WithUnitOfWork(uow styx.UnitOfWork) PersonRepository
	List() ([]models.Person, error)
	GetByID(id string) (*models.Person, error)
	Add(person *models.Person) error
	Update(id string, person *models.Person) error
	Delete(id string) error
	Count() (int, error)
	DeleteAll() error
}
