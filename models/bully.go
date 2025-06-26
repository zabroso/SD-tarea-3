package models

type Bully struct {
	ID       int
	Nodes    map[int]string // Mapa: ID → dirección IP:puerto
	LeaderID int
}
