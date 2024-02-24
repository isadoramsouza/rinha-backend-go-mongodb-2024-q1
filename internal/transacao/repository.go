package transacao

import (
	"context"
	"errors"
	"sync"
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
	clienteCollection := r.db.Database(DB_NAME).Collection("clientes")

	var c domain.Cliente
	err := clienteCollection.FindOne(ctx, bson.M{"id": t.ClienteID}).Decode(&c)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
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
		return domain.TransacaoResponse{}, LimitErr
	}

	// Converte a nova transação em uma struct UltimaTransacao
	novaTransacao := domain.UltimaTransacao{
		Tipo:        t.Tipo,
		Descricao:   t.Descricao,
		RealizadaEm: t.RealizadaEm,
		Valor:       t.Valor,
	}

	// Adiciona a nova transação ao início do array
	c.UltimasTransacoes = append([]domain.UltimaTransacao{novaTransacao}, c.UltimasTransacoes...)

	// Mantém apenas as últimas 10 transações se houver mais de 10
	if len(c.UltimasTransacoes) > 10 {
		c.UltimasTransacoes = c.UltimasTransacoes[:10]
	}

	// Atualiza o saldo e as últimas transações no documento "clientes"
	_, err = clienteCollection.UpdateOne(
		ctx,
		bson.M{"id": t.ClienteID},
		bson.M{
			"$set": bson.M{"saldo": newBalance, "ultimas_transacoes": c.UltimasTransacoes},
		},
	)
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

	cursor, err := r.db.Database(DB_NAME).Collection("clientes").Aggregate(ctx, pipeline)
	if err != nil {
		return domain.Extrato{}, err
	}
	defer cursor.Close(ctx)

	var extrato struct {
		Extrato domain.Extrato `bson:"extrato"`
	}
	if cursor.Next(ctx) {
		err := cursor.Decode(&extrato)
		if err != nil {
			return domain.Extrato{}, err
		}
	}

	// Simplified check for result presence
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
