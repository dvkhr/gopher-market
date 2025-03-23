package model

import "time"

type User struct {
	ID           int     `json:"user_id,omitempty"`         //  уникальный идентификатор пользователя
	Username     string  `json:"login,omitempty"`           // имя пользователя
	PasswordHash string  `json:"password_hash,omitempty"`   // хэш пароля пользователя
	Balance      float32 `json:"current_balance,omitempty"` // текущий баланс пользователя
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
	ID          int       `json:"id,omitempty"`          //  уникальный идентификатор заказа
	UserID      int       `json:"user_id,omitempty"`     // уникальный идентификатор пользователя
	OrderNumber string    `json:"number,omitempty"`      // номер заказа
	Accrual     float32   `json:"accrual,omitempty"`     // вознаграждение за заказ
	UploadedAt  time.Time `json:"uploaded_at,omitempty"` // время загрузки номера заказа time.RFC3339
	Status      Status    `json:"status,omitempty"`      // статус обработки заказа
}

type TType string // тип транзакции

const (
	Accrual  TType = "accrual"  // пополнение
	Withdraw TType = "withdraw" // снятие
)

type Transactions struct {
	ID               int       `json:"id,omitempty"`                //  уникальный идентификатор транзакции
	UserID           string    `json:"user_id,omitempty"`           // уникальный идентификатор пользователя
	OrderNumber      string    `json:"number,omitempty"`            // номер заказа
	Amount           float32   `json:"amount,omitempty"`            // сумма транзакции,  либо начисление (положительная, accrual), либо изъятие (отрицательная, withdrawn)
	TransactionsType TType     `json:"transactions_type,omitempty"` // тип транзакции
	UpdatedAt        time.Time `json:"updated_at,omitempty"`        // дата последнего обновления баланса time.RFC3339

}
