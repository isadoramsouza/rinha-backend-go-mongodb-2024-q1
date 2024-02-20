package transacao

import (
	"context"
	"errors"
	"time"

	"github.com/isadoramsouza/rinha-backend-go-2024-q1/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrNotFound = errors.New("cliente not found")
	LimitErr    = errors.New("limit error")
	DB_NAME     = "rinhabackenddb"
)

type Repository interface {
	SaveTransaction(ctx context.Context, t domain.Transacao) (domain.TransacaoResponse, error)
	GetBalance(ctx context.Context, id int) (domain.Cliente, error)
	GetExtrato(ctx context.Context, id int) (domain.Extrato, error)
}

type repository struct {
	db *mongo.Client
}

func NewRepository(db *mongo.Client) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) SaveTransaction(ctx context.Context, t domain.Transacao) (domain.TransacaoResponse, error) {
	transacaoCollection := r.db.Database(DB_NAME).Collection("transacoes")
	clienteCollection := r.db.Database(DB_NAME).Collection("clientes")

	// Busca o cliente para verificar seu saldo atual e limite disponível
	var c domain.Cliente
	err := clienteCollection.FindOne(ctx, bson.M{"id": t.ClienteID}).Decode(&c)
	if err != nil {
		return domain.TransacaoResponse{}, err
	}

	// Calcula o novo saldo com base no tipo de transação
	var newBalance int64
	if t.Tipo == "c" {
		newBalance = c.Saldo + int64(t.Valor)
	} else {
		newBalance = c.Saldo - int64(t.Valor)
	}

	// Verifica se a transação viola a restrição do limite disponível
	if t.Tipo == "d" && (newBalance < -c.Limite) {
		return domain.TransacaoResponse{}, LimitErr
	}

	// Atualiza o saldo do cliente e insere a transação em uma única operação de banco de dados
	update := bson.M{"$set": bson.M{"saldo": newBalance}}
	_, err = clienteCollection.UpdateOne(ctx, bson.M{"id": t.ClienteID}, update)
	if err != nil {
		return domain.TransacaoResponse{}, err
	}

	// Insere a transação na coleção de transações
	_, err = transacaoCollection.InsertOne(ctx, t)
	if err != nil {
		return domain.TransacaoResponse{}, err
	}

	// Retorna a resposta da transação
	response := domain.TransacaoResponse{
		Saldo:  newBalance,
		Limite: c.Limite,
	}

	return response, nil
}

func (r *repository) GetBalance(ctx context.Context, id int) (domain.Cliente, error) {
	collection := r.db.Database(DB_NAME).Collection("clientes")

	var c domain.Cliente
	err := collection.FindOne(ctx, bson.M{"id": id}).Decode(&c)
	if err != nil {
		return domain.Cliente{}, err
	}

	return c, nil
}

func (r *repository) GetExtrato(ctx context.Context, id int) (domain.Extrato, error) {
	pipeline := bson.A{
		bson.D{{"$match", bson.M{"cliente_id": id}}},
		bson.D{{"$sort", bson.M{"realizada_em": -1}}},
		bson.D{{"$limit", 10}},
	}

	transacaoCollection := r.db.Database(DB_NAME).Collection("transacoes")

	cur, err := transacaoCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return domain.Extrato{}, err
	}
	defer cur.Close(ctx)

	var transacoes []domain.UltimaTransacao

	for cur.Next(ctx) {
		var t domain.UltimaTransacao
		err := cur.Decode(&t)
		if err != nil {
			return domain.Extrato{}, err
		}
		transacoes = append(transacoes, t)
	}

	if err := cur.Err(); err != nil {
		return domain.Extrato{}, err
	}

	cliente, err := r.GetBalance(ctx, id)
	if err != nil {
		return domain.Extrato{}, err
	}

	extrato := domain.Extrato{
		Saldo: domain.Saldo{
			Total:       cliente.Saldo,
			DataExtrato: time.Now().UTC(),
			Limite:      cliente.Limite,
		},
		UltimasTransacoes: transacoes,
	}

	return extrato, nil
}
