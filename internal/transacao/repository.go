package transacao

import (
	"context"
	"errors"
	"time"

	"github.com/isadoramsouza/rinha-backend-go-2024-q1/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	session, err := r.db.StartSession()
	if err != nil {
		return domain.TransacaoResponse{}, err
	}
	defer session.EndSession(ctx)

	err = session.StartTransaction()
	if err != nil {
		return domain.TransacaoResponse{}, err
	}

	transacaoCollection := r.db.Database(DB_NAME).Collection("transacoes")
	clienteCollection := r.db.Database(DB_NAME).Collection("clientes")

	var c domain.Cliente
	err = clienteCollection.FindOne(ctx, bson.M{"id": t.ClienteID}).Decode(&c)
	if err != nil {
		session.AbortTransaction(ctx)
		return domain.TransacaoResponse{}, err
	}

	var newBalance int64
	if t.Tipo == "c" {
		newBalance = c.Saldo + int64(t.Valor)
	} else {
		newBalance = c.Saldo - int64(t.Valor)
	}

	if newBalance < 0 {
		session.AbortTransaction(ctx)
		return domain.TransacaoResponse{}, LimitErr
	}

	_, err = clienteCollection.UpdateOne(ctx, bson.M{"id": t.ClienteID}, bson.M{"$set": bson.M{"saldo": newBalance}})
	if err != nil {
		session.AbortTransaction(ctx)
		return domain.TransacaoResponse{}, err
	}

	_, err = transacaoCollection.InsertOne(ctx, bson.M{
		"cliente_id":   t.ClienteID,
		"valor":        t.Valor,
		"tipo":         t.Tipo,
		"descricao":    t.Descricao,
		"realizada_em": primitive.NewDateTimeFromTime(time.Now().UTC()),
	})
	if err != nil {
		session.AbortTransaction(ctx)
		return domain.TransacaoResponse{}, err
	}

	err = session.CommitTransaction(ctx)
	if err != nil {
		return domain.TransacaoResponse{}, err
	}

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
	cliente, err := r.GetBalance(ctx, id)
	if err != nil {
		return domain.Extrato{}, err
	}

	transacaoCollection := r.db.Database(DB_NAME).Collection("transacoes")

	filter := bson.M{"cliente_id": id}

	options := options.Find().SetSort(bson.D{{"realizada_em", -1}}).SetLimit(10)

	cur, err := transacaoCollection.Find(ctx, filter, options)
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
