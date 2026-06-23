package services

import "github.com/masudur-rahman/kazi-ancestry/models"

// SuggestionService is the business-logic seam for the review inbox.
type SuggestionService interface {
	List() ([]models.Suggestion, error)
	Submit(suggestion *models.Suggestion) error
	Approve(id string) error
	Reject(id string) error
	Delete(id string) error
}
