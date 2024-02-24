package transacao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/isadoramsouza/rinha-backend-go-2024-q1/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrNotFound                 = errors.New("cliente not found")
	LimitErr                    = errors.New("limit error")
	DB_NAME                     = "rinhabackenddb"
	Limites     map[int64]int64 = map[int64]int64{
		1: 100000,
		2: 80000,
		3: 1000000,
		4: 10000000,
		5: 500000,
	}
)

type Repository interface {
	SaveTransaction(ctx context.Context, t domain.Transacao) (domain.TransacaoResponse, error)
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
	clienteCollection := r.db.Database(DB_NAME).Collection("clientes")

	valorTransacao := t.Valor

	transacaoJson, err := json.Marshal(t)
	if err != nil {
		return domain.TransacaoResponse{}, err
	}

	filter := bson.M{"id": t.ClienteID}
	if t.Tipo == "d" {
		valorTransacao = -t.Valor
		filter["disponivel"] = bson.M{"$gte": t.Valor}
	}

	update := bson.D{
		{"$set", bson.D{
			{"disponivel", bson.D{
				{"$add", []interface{}{"$disponivel", valorTransacao}},
			}},
			{"saldo", bson.D{
				{"$add", []interface{}{"$saldo", valorTransacao}},
			}},
			{"ultimas_transacoes", bson.D{
				{"$concatArrays", []interface{}{[]interface{}{string(transacaoJson)}, bson.D{
					{"$slice", []interface{}{"$ultimas_transacoes", 9}},
				}}},
			}},
		}},
	}

	after := options.After
	opts := options.FindOneAndUpdateOptions{
		Projection:     bson.D{{"saldo", 1}},
		ReturnDocument: &after,
	}
	result := &domain.Result{}
	err = clienteCollection.FindOneAndUpdate(ctx, filter, mongo.Pipeline{update}, &opts).Decode(&result)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return domain.TransacaoResponse{}, LimitErr
		}
		return domain.TransacaoResponse{}, err
	}

	response := domain.TransacaoResponse{
		Saldo:  result.Saldo,
		Limite: Limites[int64(t.ClienteID)],
	}
	return response, nil
}

func (r *repository) GetExtrato(ctx context.Context, id int) (domain.Extrato, error) {
	clienteCollection := r.db.Database(DB_NAME).Collection("clientes")

	filter := bson.M{"id": id}
	projection := bson.M{"saldo": 1, "ultimas_transacoes": 1}

	var result struct {
		Saldo             int64    `json:"saldo" bson:"saldo"`
		UltimasTransacoes []string `bson:"ultimas_transacoes" json:"ultimas_transacoes"`
	}

	extrato := &result

	err := clienteCollection.FindOne(ctx, filter, &options.FindOneOptions{Projection: projection}).Decode(extrato)
	if err != nil {
		return domain.Extrato{}, err
	}

	var ultimasTransacoes []domain.UltimaTransacao

	err = json.Unmarshal([]byte(fmt.Sprintf("[%s]", strings.Join(result.UltimasTransacoes, ","))), &ultimasTransacoes)
	if err != nil {
		return domain.Extrato{}, err
	}

	response := domain.Extrato{
		Saldo: domain.Saldo{
			Total:       result.Saldo,
			Limite:      Limites[int64(id)],
			DataExtrato: time.Now().Format(time.RFC3339),
		},
		UltimasTransacoes: ultimasTransacoes,
	}

	return response, nil
}
