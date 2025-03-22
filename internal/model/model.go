package model

import "time"

type User struct {
	Id            int     `json:"user_id"`         //  уникальный идентификатор пользователя
	Username      string  `json:"login"`           // имя пользователя
	Password_hash string  `json:"password_hash"`   // хэш пароля пользователя
	Balance       float64 `json:"current_balance"` // текущий баланс пользователя
}

type Status string

const (
	StatusNew        Status = "new"        //заказ загружен в систему, но не попал в обработку
	StatusProcessing Status = "processing" //вознаграждение за заказ рассчитывается
	StatusInvalid    Status = "invalid"    // система расчёта вознаграждений отказала в расчёте
	StatusProcessed  Status = "processed"  //данные по заказу проверены и информация о расчёте успешно получена
)

type Order struct {
	Id           int       `json:"id"`          //  уникальный идентификатор заказа
	User_id      int       `json:"user_id"`     // уникальный идентификатор пользователя
	Order_number int       `json:"number"`      // номер заказа
	Accrual      float64   `json:"Accrual"`     // вознаграждение за заказ
	Uploaded_at  time.Time `json:"uploaded_at"` // время загрузки номера заказа time.RFC3339
	Status       Status    `json:"status"`      // статус обработки заказа
}

type T_type string // тип транзакции

const (
	Accrual  T_type = "accrual"  // пополнение
	Withdraw T_type = "withdraw" // снятие
)

type Transactions struct {
	Id                int       `json:"id"`                //  уникальный идентификатор транзакции
	User_id           string    `json:"user_id"`           // уникальный идентификатор пользователя
	Order_number      int       `json:"number"`            // номер заказа
	Amount            float64   `json:"Amount"`            // сумма транзакции,  либо начисление (положительная, accrual), либо изъятие (отрицательная, withdrawn)
	Transactions_type T_type    `json:"Transactions_type"` // тип транзакции
	Updated_at        time.Time `json:"updated_at"`        // дата последнего обновления баланса time.RFC3339

}
