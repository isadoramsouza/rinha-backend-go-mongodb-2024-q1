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

	// Define as opções para retornar o documento após a atualização
	options := options.FindOneAndUpdate().SetReturnDocument(options.After)

	// Define o filtro para encontrar o cliente
	filter := bson.M{"id": t.ClienteID}

	// Define a atualização com base no tipo de transação
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

	// Realiza a atualização atômica do saldo do cliente e retorna o documento atualizado
	var updatedCliente domain.Cliente
	err := clienteCollection.FindOneAndUpdate(ctx, filter, update, options).Decode(&updatedCliente)
	if err != nil {
		return domain.TransacaoResponse{}, err
	}

	// Verifica se a transação viola a restrição do limite disponível
	if t.Tipo == "d" && (updatedCliente.Saldo < -updatedCliente.Limite) {
		return domain.TransacaoResponse{}, LimitErr
	}

	// Insere a transação na coleção de transações
	_, err = transacaoCollection.InsertOne(ctx, t)
	if err != nil {
		return domain.TransacaoResponse{}, err
	}

	// Retorna a resposta da transação
	response := domain.TransacaoResponse{
		Saldo:  updatedCliente.Saldo,
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
