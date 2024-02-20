package transacao

import (
	"context"
	"errors"
	"sync"
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

var mutex sync.Mutex

type Repository interface {
	SaveTransaction(ctx context.Context, t domain.Transacao) (domain.TransacaoResponse, error)
	GetExtrato(ctx context.Context, id int) (domain.Extrato, error)
}

type repository struct {
	db   *mongo.Client
	lock sync.Mutex
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

	clienteCollection := r.db.Database(DB_NAME).Collection("clientes")

	var c domain.Cliente
	err = clienteCollection.FindOne(ctx, bson.M{"id": t.ClienteID}).Decode(&c)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		session.AbortTransaction(ctx)
		return domain.TransacaoResponse{}, err
	}

	// Calcula o novo saldo
	var newBalance int64
	if t.Tipo == "c" {
		newBalance = c.Saldo + int64(t.Valor)
	} else {
		newBalance = c.Saldo - int64(t.Valor)
	}

	// Verifica se o saldo ultrapassa o limite
	if (newBalance + int64(c.Limite)) < 0 {
		session.AbortTransaction(ctx)
		return domain.TransacaoResponse{}, LimitErr
	}

	// Atualiza o saldo e adiciona a transação ao array ultimas_transacoes
	filter := bson.M{"id": t.ClienteID}
	update := bson.M{
		"$set": bson.M{"saldo": newBalance},
		"$push": bson.M{"ultimas_transacoes": bson.M{
			"tipo":         t.Tipo,
			"descricao":    t.Descricao,
			"realizada_em": t.RealizadaEm,
			"valor":        t.Valor,
		}},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	err = clienteCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&c)
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

func (r *repository) GetExtrato(ctx context.Context, id int) (domain.Extrato, error) {
	// Define a data do extrato
	dataExtrato := time.Now().UTC()

	// Define a pipeline de agregação para buscar as últimas transações e informações do cliente
	pipeline := []bson.M{
		{"$match": bson.M{"id": id}},
		{"$lookup": bson.M{
			"from":         "clientes",
			"localField":   "id",
			"foreignField": "id",
			"as":           "cliente",
		}},
		{"$unwind": "$cliente"},
		{"$project": bson.M{
			"saldo": bson.M{
				"total":        "$cliente.saldo",
				"data_extrato": bson.M{"$toDate": dataExtrato},
				"limite":       "$cliente.limite",
			},
			"ultimas_transacoes": "$cliente.ultimas_transacoes",
		}},
		{"$unwind": bson.M{"path": "$ultimas_transacoes", "preserveNullAndEmptyArrays": true}}, // Unwind das transações, preservando arrays nulos ou vazios
		{"$sort": bson.M{"ultimas_transacoes.realizada_em": -1}},                               // Ordenar as transações por data em ordem decrescente
		{"$group": bson.M{
			"_id":                "$_id",
			"saldo":              bson.M{"$first": "$saldo"},
			"ultimas_transacoes": bson.M{"$push": "$ultimas_transacoes"},
		}},
		{"$project": bson.M{
			"extrato": bson.M{
				"saldo":              "$saldo",
				"ultimas_transacoes": bson.M{"$slice": []interface{}{"$ultimas_transacoes", 10}}, // Limitar a 10 transações
			},
		}},
	}

	// Executa a agregação
	cursor, err := r.db.Database(DB_NAME).Collection("clientes").Aggregate(ctx, pipeline)
	if err != nil {
		return domain.Extrato{}, err
	}
	defer cursor.Close(ctx)

	// Decodifica o resultado da agregação
	var extrato struct {
		Extrato domain.Extrato `bson:"extrato"`
	}
	if cursor.Next(ctx) {
		err := cursor.Decode(&extrato)
		if err != nil {
			return domain.Extrato{}, err
		}
	}

	// Se não houver resultados, retorna um Extrato com os valores padrão
	if extrato.Extrato.UltimasTransacoes == nil {
		return domain.Extrato{
			Saldo: domain.Saldo{
				Total:       extrato.Extrato.Saldo.Total,
				DataExtrato: extrato.Extrato.Saldo.DataExtrato,
				Limite:      extrato.Extrato.Saldo.Limite,
			},
			UltimasTransacoes: nil,
		}, nil
	}

	return extrato.Extrato, nil
}
