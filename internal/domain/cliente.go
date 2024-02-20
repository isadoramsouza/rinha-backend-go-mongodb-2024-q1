package domain

type Cliente struct {
	ID                int               `json:"id" bson:"id"`
	Limite            int64             `json:"limite" bson:"limite"`
	Saldo             int64             `json:"saldo" bson:"saldo"`
	UltimasTransacoes []UltimaTransacao `json:"ultimas_transacoes" bson:"ultimas_transacoes"`
}
