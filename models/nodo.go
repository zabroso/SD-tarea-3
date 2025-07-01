package models

import "sync"

type Event struct {
	Id    int
	Value string
	Nodo  string
}

type Nodo struct {
	ID             string  `json:"id"`
	IsPrimary      bool    `json:"is_primary"`
	LastMessage    string  `json:"last_message"`
	SequenceNumber int     `json:"sequence_number"`
	EventLog       []Event `json:"event_log"` // Antes era []string
	Port           int     `json:"port"`
	Mu             sync.Mutex
}
