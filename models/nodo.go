package models

import "sync"

type Nodo struct {
	ID             string   `json:"id"`
	IsPrimary      bool     `json:"is_primary"`
	LastMessage    string   `json:"last_message"`
	SequenceNumber int      `json:"sequence_number"`
	EventLog       []string `json:"event_log"`
	Port           int      `json:"port"`
	mu             sync.Mutex
}
