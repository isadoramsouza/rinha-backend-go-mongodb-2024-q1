package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isadoramsouza/rinha-backend-go-mongodb-2024-q1/cmd/api/routes"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	DB_NAME = "rinhabackenddb"
)

func main() {

	mongoURI := "mongodb://localhost:27017/rinhabackenddb"

	ctx := context.Background()

	opts := options.Client().SetTimeout(time.Duration(time.Second * 6)).SetMaxPoolSize(uint64(350)).SetMinPoolSize(uint64(150)).ApplyURI(mongoURI)
	db, err := mongo.Connect(ctx, opts)
	CheckError(err)

	clientesCollection := db.Database(DB_NAME).Collection("clientes")

	_, err = clientesCollection.Indexes().CreateOne(
		context.TODO(),
		mongo.IndexModel{Keys: bson.D{
			{Key: "id", Value: 1},
			{Key: "disponivel", Value: 1},
			{Key: "saldo", Value: 1},
			{Key: "ultimas_transacoes", Value: -1},
		}},
	)

	CheckError(err)

	defer db.Disconnect(ctx)

	fmt.Println("Connected to MongoDB!")

	eng := gin.New()

	router := routes.NewRouter(eng, db)
	router.MapRoutes()

	if err := eng.Run(); err != nil {
		panic(err)
	}
}

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}
