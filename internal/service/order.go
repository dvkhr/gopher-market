package service

import (
	"errors"
	"gopher-market/internal/logging"
	"gopher-market/internal/model"

	"github.com/EClaesson/go-luhn"
)

var (
	ErrInvalidFormat = errors.New("invalid order number format")
	ErrInvalidNumber = errors.New("invalid order number")
)

func (s *Service) CheckOrder(orderNumber, username string) error {
	if !IsNumeric(orderNumber) {
		return ErrInvalidFormat
	}
	isValid, err := luhn.IsValid(orderNumber)
	if err != nil {
		return err
	}

	if !isValid {
		return ErrInvalidNumber
	}

	order, err := s.Repo.GetOrderByNumber(orderNumber)
	if err == nil {
		user, _ := s.Repo.GetUserByOrderNumber(orderNumber)
		if user.Username == username {
			logging.Logg.Info("Order already uploaded by the user", "order_id", order.ID)
			return errors.New("the order was uploaded by the user")
		}
		logging.Logg.Warn("Order uploaded by another user", "order_id", order.ID)
		return errors.New("order number already uploaded by another user")
	}
	return nil
}
func (s *Service) UploadOrder(userID int, orderNumber string) error {
	_, err := s.Repo.CreateOrder(userID, orderNumber)
	return err
}
func (s *Service) GetOrders(userID int) ([]model.Order, error) {
	return s.Repo.GetOrders(userID)
}
