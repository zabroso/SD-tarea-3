package models

type Bully struct {
	ID       int
	Peers    map[int]string // Mapa: ID → dirección IP:puerto
	LeaderID int
	IsLeader bool
}
