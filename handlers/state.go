package handlers

import (
	"SD-Tarea-3/models"
	"strconv"
)

var Estado *models.Nodo
var Nodes map[int]string
var PrimaryNodeID int

func SetEstado(newEstado *models.Nodo) {
	Estado = newEstado
}

func SetNodes(newNodes map[int]string) {
	Nodes = newNodes
}

func NewPrimary() {
	Estado.IsPrimary = true
	newPrimaryNodeID, _ := strconv.Atoi(Estado.ID)
	PrimaryNodeID = newPrimaryNodeID
}
