package models

type AccrualResponse struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"` // REGISTERED, PROCESSING, INVALID, PROCESSED
	Accrual *float64 `json:"accrual,omitempty"`
}
