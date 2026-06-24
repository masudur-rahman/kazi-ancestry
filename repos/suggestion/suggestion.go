package suggestion

import (
	"context"

	"github.com/masudur-rahman/kazi-ancestry/infra/logr"
	"github.com/masudur-rahman/kazi-ancestry/models"
	"github.com/masudur-rahman/kazi-ancestry/repos"

	"github.com/masudur-rahman/styx"
	isql "github.com/masudur-rahman/styx/sql"
)

type SQLSuggestionRepository struct {
	db     isql.Engine
	logger logr.Logger
}

func NewSQLSuggestionRepository(db isql.Engine, logger logr.Logger) *SQLSuggestionRepository {
	return &SQLSuggestionRepository{
		db:     db.Table(models.Suggestion{}.TableName()),
		logger: logger,
	}
}

func (r *SQLSuggestionRepository) WithUnitOfWork(uow styx.UnitOfWork) repos.SuggestionRepository {
	return &SQLSuggestionRepository{
		db:     uow.SQL.Table(models.Suggestion{}.TableName()),
		logger: r.logger,
	}
}

func (r *SQLSuggestionRepository) List() ([]models.Suggestion, error) {
	ctx := context.Background()
	var out []models.Suggestion
	if err := r.db.FindMany(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListBySubmitter returns the suggestions submitted by a given user (any status).
func (r *SQLSuggestionRepository) ListBySubmitter(email string) ([]models.Suggestion, error) {
	ctx := context.Background()
	var out []models.Suggestion
	if err := r.db.Where("submitted_by=?", email).FindMany(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SQLSuggestionRepository) Add(s *models.Suggestion) error {
	r.logger.Infow("add suggestion", "id", s.ID, "person", s.PersonID)
	ctx := context.Background()
	_, err := r.db.InsertOne(ctx, s)
	return err
}

func (r *SQLSuggestionRepository) UpdateStatus(id, status string) error {
	r.logger.Infow("update suggestion status", "id", id, "status", status)
	ctx := context.Background()
	return r.db.ID(id).MustCols("status").UpdateOne(ctx, models.Suggestion{Status: status})
}

func (r *SQLSuggestionRepository) Delete(id string) error {
	ctx := context.Background()
	return r.db.ID(id).DeleteOne(ctx)
}
