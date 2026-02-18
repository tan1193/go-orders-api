package model

import "time"

const (
	StatusCreated    = "created"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
)

type Order struct {
	ID           string    `json:"id"`
	CustomerName string    `json:"customer_name"`
	Amount       int       `json:"amount"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}
