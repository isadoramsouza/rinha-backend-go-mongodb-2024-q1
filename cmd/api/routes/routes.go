package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/isadoramsouza/rinha-backend-go-mongodb-2024-q1/cmd/api/handler"
	"github.com/isadoramsouza/rinha-backend-go-mongodb-2024-q1/internal/transacao"
	"go.mongodb.org/mongo-driver/mongo"
)

type Router interface {
	MapRoutes()
}

type router struct {
	eng *gin.Engine
	rg  *gin.RouterGroup
	db  *mongo.Client
}

func NewRouter(eng *gin.Engine, db *mongo.Client) Router {
	return &router{eng: eng, db: db}
}

func (r *router) MapRoutes() {
	r.setGroup()
	r.buildRoutes()
}

func (r *router) setGroup() {
	r.rg = r.eng.Group("")
}

func (r *router) buildRoutes() {
	repo := transacao.NewRepository(r.db)
	service := transacao.NewService(repo)
	handler := handler.NewTransacao(service)
	r.rg.POST("/clientes/:id/transacoes", handler.CreateTransaction())
	r.rg.GET("/clientes/:id/extrato", handler.GetExtrato())
}
