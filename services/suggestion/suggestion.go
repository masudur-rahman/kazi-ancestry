package suggestion

import (
	"time"

	"github.com/google/uuid"

	"github.com/masudur-rahman/kazi-ancestry/models"
	"github.com/masudur-rahman/kazi-ancestry/repos"
)

type suggestionService struct {
	repo repos.SuggestionRepository
}

func NewSuggestionService(repo repos.SuggestionRepository) *suggestionService {
	return &suggestionService{repo: repo}
}

func (s *suggestionService) List() ([]models.Suggestion, error) { return s.repo.List() }

func (s *suggestionService) Submit(sug *models.Suggestion) error {
	if sug.ID == "" {
		sug.ID = uuid.NewString()
	}
	if sug.Status == "" {
		sug.Status = "pending"
	}
	if sug.CreatedAt == 0 {
		sug.CreatedAt = time.Now().Unix()
	}
	return s.repo.Add(sug)
}

func (s *suggestionService) Approve(id string) error { return s.repo.UpdateStatus(id, "approved") }
func (s *suggestionService) Reject(id string) error  { return s.repo.UpdateStatus(id, "rejected") }
func (s *suggestionService) Delete(id string) error  { return s.repo.Delete(id) }
