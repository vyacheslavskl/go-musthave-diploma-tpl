package models

import "time"

type Order struct {
	ID        string
	UserID    string
	Number    string
	Status    string
	Accrual   *float64
	CreatedAt time.Time
}
