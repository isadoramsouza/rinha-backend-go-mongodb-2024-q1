package domain

type Cliente struct {
	ID     int   `json:"id"`
	Limite int64 `json:"limite"`
	Saldo  int64 `json:"saldo"`
}
