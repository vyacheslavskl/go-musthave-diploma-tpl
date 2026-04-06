package models

import "time"

type Order struct {
	OrderID   string
	UserID    string
	Number    string
	Status    string //  NEW, PROCESSING, INVALID, PROCESSED
	Accrual   *float64
	CreatedAt time.Time
}
