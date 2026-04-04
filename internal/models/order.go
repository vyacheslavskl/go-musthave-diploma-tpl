package models

import "time"

type Order struct {
	OrderID   string
	UserID    string
	Number    string
	Status    string
	Accrual   *float64
	CreatedAt time.Time
}
