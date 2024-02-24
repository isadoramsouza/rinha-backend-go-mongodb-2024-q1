package main

import (
	"context"
	"fmt"

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

	//mongoURI := os.Getenv("MONGODB_URI")
	mongoURI := "mongodb://localhost:27017/rinhabackenddb?socketTimeoutMS=360000&connectTimeoutMS=360000&maxPoolSize=10&minPoolSize=5"

	db, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	CheckError(err)

	ctx := context.Background()
	err = db.Connect(ctx)
	CheckError(err)

	defer db.Disconnect(ctx)

	fmt.Println("Connected to MongoDB!")

	clientesCollection := db.Database(DB_NAME).Collection("clientes")
	clientesCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.M{"id": 1}},
		{Keys: bson.M{"disponivel": 1}},
		{Keys: bson.M{"saldo": 1}},
		{Keys: bson.M{"ultimas_transacoes": -1}},
	})
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
