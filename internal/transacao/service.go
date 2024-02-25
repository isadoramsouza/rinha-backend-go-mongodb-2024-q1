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
}

func NewService(r Repository) Service {
	return &transacaoService{
		repository: r,
	}
}

func (s *transacaoService) CreateTransaction(ctx context.Context, t domain.Transacao) (domain.TransacaoResponse, error) {
	responseChan := make(chan domain.TransacaoResponse)
	errChan := make(chan error)

	go func() {
		response, err := s.repository.SaveTransaction(ctx, t)
		if err != nil {
			errChan <- err
			return
		}
		responseChan <- response
	}()

	select {
	case response := <-responseChan:
		return response, nil
	case err := <-errChan:
		return domain.TransacaoResponse{}, err
	}
}

func (s *transacaoService) GetExtrato(ctx context.Context, id int) (domain.Extrato, error) {
	extratoChan := make(chan domain.Extrato)
	errChan := make(chan error)

	go func() {
		extrato, err := s.repository.GetExtrato(ctx, id)
		if err != nil {
			errChan <- err
			return
		}
		extratoChan <- extrato
	}()

	select {
	case extrato := <-extratoChan:
		return extrato, nil
	case err := <-errChan:
		return domain.Extrato{}, err
	}
}
