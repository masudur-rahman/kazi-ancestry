package all

import (
	"sync"

	"github.com/masudur-rahman/kazi-ancestry/infra/logr"
	personrepo "github.com/masudur-rahman/kazi-ancestry/repos/person"
	suggestionrepo "github.com/masudur-rahman/kazi-ancestry/repos/suggestion"
	"github.com/masudur-rahman/kazi-ancestry/services"
	personsvc "github.com/masudur-rahman/kazi-ancestry/services/person"
	suggestionsvc "github.com/masudur-rahman/kazi-ancestry/services/suggestion"

	"github.com/masudur-rahman/styx"
)

// Services aggregates the application's service layer.
type Services struct {
	Person     services.PersonService
	Suggestion services.SuggestionService
}

var (
	svc   *Services
	svcMu sync.RWMutex
)

// GetServices returns the initialized service set.
func GetServices() *Services {
	svcMu.RLock()
	defer svcMu.RUnlock()
	return svc
}

// InitiateSQLServices wires repositories to services over the given unit of work.
func InitiateSQLServices(uow styx.UnitOfWork, logger logr.Logger) {
	personRepo := personrepo.NewSQLPersonRepository(uow.SQL, logger)
	suggestionRepo := suggestionrepo.NewSQLSuggestionRepository(uow.SQL, logger)

	svcMu.Lock()
	svc = &Services{
		Person:     personsvc.NewPersonService(personRepo),
		Suggestion: suggestionsvc.NewSuggestionService(suggestionRepo),
	}
	svcMu.Unlock()
}
