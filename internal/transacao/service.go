package transacao

import (
	"context"

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
	responseChan := make(chan domain.TransacaoResponse, 1) // Use um buffer para evitar bloqueios desnecessários
	errChan := make(chan error, 1)                         // Use um buffer para evitar bloqueios desnecessários

	select {
	case s.semaphore <- struct{}{}:
		go func() {
			defer func() {
				<-s.semaphore
			}()

			response, err := s.repository.SaveTransaction(ctx, t)
			if err != nil {
				errChan <- err
				return
			}
			responseChan <- response
		}()
	case <-ctx.Done():
		return domain.TransacaoResponse{}, ctx.Err() // Trate o cancelamento do contexto
	}

	select {
	case response := <-responseChan:
		return response, nil
	case err := <-errChan:
		return domain.TransacaoResponse{}, err
	}
}

func (s *transacaoService) GetExtrato(ctx context.Context, id int) (domain.Extrato, error) {
	extratoChan := make(chan domain.Extrato, 1) // Use um buffer para evitar bloqueios desnecessários
	errChan := make(chan error, 1)              // Use um buffer para evitar bloqueios desnecessários

	select {
	case s.semaphore <- struct{}{}:
		go func() {
			defer func() {
				<-s.semaphore
			}()

			extrato, err := s.repository.GetExtrato(ctx, id)
			if err != nil {
				errChan <- err
				return
			}
			extratoChan <- extrato
		}()
	case <-ctx.Done():
		return domain.Extrato{}, ctx.Err() // Trate o cancelamento do contexto
	}

	select {
	case extrato := <-extratoChan:
		return extrato, nil
	case err := <-errChan:
		return domain.Extrato{}, err
	}
}
