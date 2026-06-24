package repos

import (
	"github.com/masudur-rahman/kazi-ancestry/models"

	"github.com/masudur-rahman/styx"
)

// SuggestionRepository is the data-access seam for proposed edits.
type SuggestionRepository interface {
	WithUnitOfWork(uow styx.UnitOfWork) SuggestionRepository
	List() ([]models.Suggestion, error)
	ListBySubmitter(email string) ([]models.Suggestion, error)
	Add(suggestion *models.Suggestion) error
	UpdateStatus(id, status string) error
	Delete(id string) error
}
