package handlers

import (
	"SD-Tarea-3/models"
)

var Estado *models.Nodo
var Nodes map[int]string

func SetEstado(newEstado *models.Nodo) {
	Estado = newEstado
}

func SetNodes(newNodes map[int]string) {
	Nodes = newNodes
}

func NewPrimary() {
	Estado.IsPrimary = true
}
