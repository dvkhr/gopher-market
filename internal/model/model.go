package model

import "time"

type User struct {
	ID           int     `json:"user_id"`         //  уникальный идентификатор пользователя
	Username     string  `json:"login"`           // имя пользователя
	PasswordHash string  `json:"password_hash"`   // хэш пароля пользователя
	Balance      float64 `json:"current_balance"` // текущий баланс пользователя
}

type Status string

const (
	StatusNew        Status = "new"        //заказ загружен в систему, но не попал в обработку
	StatusProcessing Status = "processing" //вознаграждение за заказ рассчитывается
	StatusInvalid    Status = "invalid"    // система расчёта вознаграждений отказала в расчёте
	StatusProcessed  Status = "processed"  //данные по заказу проверены и информация о расчёте успешно получена
)

type Order struct {
	ID          int       `json:"id"`          //  уникальный идентификатор заказа
	UserID      int       `json:"user_id"`     // уникальный идентификатор пользователя
	OrderNumber int       `json:"number"`      // номер заказа
	Accrual     float64   `json:"Accrual"`     // вознаграждение за заказ
	UploadedAt  time.Time `json:"uploaded_at"` // время загрузки номера заказа time.RFC3339
	Status      Status    `json:"status"`      // статус обработки заказа
}

type TType string // тип транзакции

const (
	Accrual  TType = "accrual"  // пополнение
	Withdraw TType = "withdraw" // снятие
)

type Transactions struct {
	ID               int       `json:"id"`                //  уникальный идентификатор транзакции
	UserID           string    `json:"user_id"`           // уникальный идентификатор пользователя
	OrderNumber      int       `json:"number"`            // номер заказа
	Amount           float64   `json:"Amount"`            // сумма транзакции,  либо начисление (положительная, accrual), либо изъятие (отрицательная, withdrawn)
	TransactionsType TType     `json:"Transactions_type"` // тип транзакции
	UpdatedAt        time.Time `json:"updated_at"`        // дата последнего обновления баланса time.RFC3339

}
