package person

import (
	"context"

	"github.com/masudur-rahman/kazi-ancestry/infra/logr"
	"github.com/masudur-rahman/kazi-ancestry/models"
	"github.com/masudur-rahman/kazi-ancestry/repos"

	"github.com/masudur-rahman/styx"
	isql "github.com/masudur-rahman/styx/sql"
)

type SQLPersonRepository struct {
	db     isql.Engine
	logger logr.Logger
}

func NewSQLPersonRepository(db isql.Engine, logger logr.Logger) *SQLPersonRepository {
	return &SQLPersonRepository{
		db:     db.Table(models.Person{}.TableName()),
		logger: logger,
	}
}

func (r *SQLPersonRepository) WithUnitOfWork(uow styx.UnitOfWork) repos.PersonRepository {
	return &SQLPersonRepository{
		db:     uow.SQL.Table(models.Person{}.TableName()),
		logger: r.logger,
	}
}

func (r *SQLPersonRepository) List() ([]models.Person, error) {
	ctx := context.Background()
	var people []models.Person
	if err := r.db.FindMany(ctx, &people); err != nil {
		return nil, err
	}
	return people, nil
}

func (r *SQLPersonRepository) GetByID(id string) (*models.Person, error) {
	ctx := context.Background()
	var p models.Person
	found, err := r.db.ID(id).FindOne(ctx, &p)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, models.ErrPersonNotFound{ID: id}
	}
	return &p, nil
}

func (r *SQLPersonRepository) Add(person *models.Person) error {
	r.logger.Infow("add person", "id", person.ID)
	ctx := context.Background()
	_, err := r.db.InsertOne(ctx, person)
	return err
}

func (r *SQLPersonRepository) Update(id string, person *models.Person) error {
	r.logger.Infow("update person", "id", id)
	ctx := context.Background()
	return r.db.ID(id).UpdateOne(ctx, *person)
}

func (r *SQLPersonRepository) Delete(id string) error {
	r.logger.Infow("delete person", "id", id)
	ctx := context.Background()
	return r.db.ID(id).DeleteOne(ctx)
}

func (r *SQLPersonRepository) Count() (int, error) {
	people, err := r.List()
	if err != nil {
		return 0, err
	}
	return len(people), nil
}

// DeleteAll clears the table (used by reseed to regenerate ids).
func (r *SQLPersonRepository) DeleteAll() error {
	ctx := context.Background()
	_, err := r.db.Exec(ctx, "DELETE FROM "+models.Person{}.TableName())
	return err
}
