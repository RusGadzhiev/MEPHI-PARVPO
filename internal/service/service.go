package service

import (
	"context"
	"fmt"

	"github.com/RusGadzhiev/MEPHI-PARVPO/pkg/logger"
)

var (
	ErrSoldOut = fmt.Errorf("sold out")
	ErrNoConcert = fmt.Errorf("no such concert")
	ErrUserRegistred = fmt.Errorf("user is already registred")
)

type Concert struct {
	Name              string
	QtyOccupiedPlaces uint16
	PlacesQty         uint16
	SoldOut           bool
}

type Record struct {
	Username string
	Concert  string
}

type Storage interface {
	GetConcerts(ctx context.Context) ([]Concert, error)
	GetConcert(ctx context.Context, name string) (*Concert, error)
	AddRecord(ctx context.Context, record Record) error
	IsRecordExist(ctx context.Context, record Record) bool
}

type service struct {
	repo Storage
}

func NewService(repo Storage) *service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetConcerts(ctx context.Context) ([]Concert, error) {
	return s.repo.GetConcerts(ctx)
}

func (s *service) AddRecord(ctx context.Context, record Record) error {
	concert, err := s.repo.GetConcert(ctx, record.Concert)
	if err != nil {
		logger.Info("AddRecord err (get concert): %s", err.Error())
		return ErrNoConcert
	}

	if concert.SoldOut {
		logger.Infof("Sold out on the concert: %s", concert.Name)
		return ErrSoldOut
	}

	exist := s.repo.IsRecordExist(ctx, record)
	if exist {
		return ErrUserRegistred
	}

	err = s.repo.AddRecord(ctx, record)
	if err != nil {
		return fmt.Errorf("service add record error: %w", err)
	}
	return nil

}
