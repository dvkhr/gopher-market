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
	StatusNew        Status = "NEW"        //заказ загружен в систему, но не попал в обработку
	StatusRegistred  Status = "REGISTERED" //заказ загружен в систему, но не попал в обработку
	StatusProcessing Status = "PROCESSING" //вознаграждение за заказ рассчитывается
	StatusInvalid    Status = "INVALID"    // система расчёта вознаграждений отказала в расчёте
	StatusProcessed  Status = "PROCESSED"  //данные по заказу проверены и информация о расчёте успешно получена
)

type Order struct {
	ID          int       `json:"id"`          //  уникальный идентификатор заказа
	UserID      int       `json:"user_id"`     // уникальный идентификатор пользователя
	OrderNumber string    `json:"number"`      // номер заказа
	Accrual     float64   `json:"accrual"`     // вознаграждение за заказ
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
	OrderNumber      string    `json:"number"`            // номер заказа
	Amount           float64   `json:"amount"`            // сумма транзакции,  либо начисление (положительная, accrual), либо изъятие (отрицательная, withdrawn)
	TransactionsType TType     `json:"transactions_type"` // тип транзакции
	UpdatedAt        time.Time `json:"updated_at"`        // дата последнего обновления баланса time.RFC3339

}
