package domain

import "time"

type Extrato struct {
	Saldo             Saldo             `json:"saldo" bson:"saldo"`
	UltimasTransacoes []UltimaTransacao `json:"ultimas_transacoes" bson:"ulltimas_transacoes"`
}

type Saldo struct {
	Total       int64     `json:"total" bson:"total"`
	DataExtrato time.Time `json:"data_extrato" bson:"data_extrato"`
	Limite      int64     `json:"limite" bson:"limite"`
}

type UltimaTransacao struct {
	Tipo        string    `json:"tipo" bson:"tipo"`
	Descricao   string    `json:"descricao" bson:"descricao"`
	Valor       int64     `json:"valor" bson:"valor"`
	RealizadaEm time.Time `json:"realizada_em" bson:"realizada_em"`
}
