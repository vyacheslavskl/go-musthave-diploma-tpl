package apperrors

import "errors"

var (
	ErrOrderAlreadyExists          = errors.New("order already exists")
	ErrOrderAlreadyExistsForUser   = errors.New("order already exists for user")
	ErrOrderAlreadyExistsOtherUser = errors.New("order already exists for another user")
	ErrInvalidOrderNumber          = errors.New("invalid order number")
	ErrInvalidSum                  = errors.New("invalid sum")
	ErrInsufficientFunds           = errors.New("insufficient funds")
)
