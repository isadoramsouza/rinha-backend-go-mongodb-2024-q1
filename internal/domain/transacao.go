package domain

import "time"

type Transacao struct {
	ID          int       `json:"id" bson:"id"`
	Tipo        string    `json:"tipo" bson:"tipo"`
	Descricao   string    `json:"descricao" bson:"descricao"`
	Valor       int64     `json:"valor" bson:"valor"`
	ClienteID   int       `json:"cliente_id" bson:"cliente_id"`
	RealizadaEm time.Time `json:"realizada_em" bson:"realizada_em"`
}

type TransacaoResponse struct {
	Limite int64 `json:"limite" bson:"limite"`
	Saldo  int64 `json:"saldo" bson:"saldo"`
}
