package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isadoramsouza/rinha-backend-go-2024-q1/cmd/api/routes"
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
	clientesCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.M{"id": 1}},
		{Keys: bson.M{"disponivel": 1}},
		{Keys: bson.M{"saldo": 1}},
		{Keys: bson.M{"ultimas_transacoes": -1}},
	})

	CheckError(err)

	defer db.Disconnect(ctx)

	fmt.Println("Connected to MongoDB!")

	eng := gin.Default()

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
