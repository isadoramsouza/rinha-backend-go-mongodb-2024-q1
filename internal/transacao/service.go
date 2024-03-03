package transacao

import (
	"context"
	"sync"
	"time"

	"github.com/isadoramsouza/rinha-backend-go-2024-q1/internal/domain"
)

type Service interface {
	CreateTransaction(ctx context.Context, t domain.Transacao) (domain.TransacaoResponse, error)
	GetExtrato(ctx context.Context, id int) (domain.Extrato, error)
}

type transacaoService struct {
	repository Repository
	semaphore  chan struct{}
}

func NewService(r Repository) Service {
	return &transacaoService{
		repository: r,
		semaphore:  make(chan struct{}, 20),
	}
}

func (s *transacaoService) CreateTransaction(ctx context.Context, t domain.Transacao) (domain.TransacaoResponse, error) {
	var (
		response domain.TransacaoResponse
		err      error
		wg       sync.WaitGroup
	)

	select {
	case s.semaphore <- struct{}{}:
		wg.Add(1)
		defer func() {
			<-s.semaphore
			wg.Done()
		}()

		ctx, cancel := context.WithTimeout(ctx, time.Second*5) // Defina um tempo limite
		defer cancel()

		go func() {
			defer wg.Done()
			response, err = s.repository.SaveTransaction(ctx, t)
		}()
	case <-ctx.Done():
		return domain.TransacaoResponse{}, ctx.Err()
	}

	wg.Wait()

	return response, err
}

func (s *transacaoService) GetExtrato(ctx context.Context, id int) (domain.Extrato, error) {
	var (
		extrato domain.Extrato
		err     error
		wg      sync.WaitGroup
	)

	select {
	case s.semaphore <- struct{}{}:
		wg.Add(1)
		defer func() {
			<-s.semaphore
			wg.Done()
		}()

		ctx, cancel := context.WithTimeout(ctx, time.Second*5) // Defina um tempo limite
		defer cancel()

		go func() {
			defer wg.Done()
			extrato, err = s.repository.GetExtrato(ctx, id)
		}()
	case <-ctx.Done():
		return domain.Extrato{}, ctx.Err()
	}

	wg.Wait()

	return extrato, err
}
