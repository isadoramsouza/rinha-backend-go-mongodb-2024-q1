package transacao

import (
	"context"
	"errors"
	"time"

	"github.com/isadoramsouza/rinha-backend-go-2024-q1/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	var update bson.M
	if t.Tipo == "c" {
		update = bson.M{
			"$inc": bson.M{"saldo": t.Valor},
		}
	} else {
		update = bson.M{
			"$inc": bson.M{"saldo": -t.Valor},
		}
	}

	options := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updatedCliente domain.Cliente
	err := clienteCollection.FindOneAndUpdate(ctx, bson.M{"id": t.ClienteID}, update, options).Decode(&updatedCliente)
	if err != nil {
		return domain.TransacaoResponse{}, err
	}

	newBalance := updatedCliente.Saldo
	if (newBalance + int64(updatedCliente.Limite)) < 0 {
		return domain.TransacaoResponse{}, LimitErr
	}

	_, err = transacaoCollection.InsertOne(ctx, t)
	if err != nil {
		return domain.TransacaoResponse{}, err
	}

	response := domain.TransacaoResponse{
		Saldo:  newBalance,
		Limite: updatedCliente.Limite,
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
