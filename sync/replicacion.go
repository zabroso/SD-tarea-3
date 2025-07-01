package replicacion

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	"SD-Tarea-3/handlers"
	"SD-Tarea-3/models"
	"SD-Tarea-3/proto"

	"google.golang.org/grpc"
)

// Registra eventos en el nodo y los replica a otros nodos
// Este método se invoca cuando un nodo recibe una pelota y debe registrar los eventos
func RegistrarYReplicarEventos(destinoNodo int) {
	nodoStr := strconv.Itoa(destinoNodo)

	agregarEvento("Jugando con la pelota", nodoStr)
	agregarEvento("Se le cayó la pelota", nodoStr)
	agregarEvento("Buscando la pelota", nodoStr)

	for _, direccion := range handlers.Nodes {
		if direccion != handlers.Nodes[handlers.GetNodeID()] {
			go func(direccion string) {
				conn, err := grpc.Dial(direccion, grpc.WithInsecure())
				if err != nil {
					log.Printf("Error al conectar con %s para replicación: %v", direccion, err)
					return
				}
				defer conn.Close()

				client := proto.NewNodoServiceClient(conn)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				logs := &proto.Logs{
					SequenceNumber: int32(handlers.Estado.SequenceNumber),
					EventLog:       []*proto.Event{},
				}
				eventos := handlers.Estado.EventLog
				start := len(eventos) - 3

				if start < 0 {
					start = 0
				}

				for _, e := range eventos[start:] {
					logs.EventLog = append(logs.EventLog, &proto.Event{
						Id:    int32(e.Id),
						Value: e.Value,
						Nodo:  e.Nodo,
					})
				}

				resp, err := client.Replicar(ctx, logs)
				if err != nil {
					log.Printf("Error replicando logs a %s: %v", direccion, err)
				} else {
					log.Printf("Logs replicados exitosamente a %s. SequenceNumber remoto: %d", direccion, resp.GetSequenceNumber())
				}
			}(direccion)
		}
	}
}

// Agrega un evento al registro de eventos del nodo y actualiza el sequence number
func agregarEvento(mensaje string, nodo string) {
	handlers.Estado.Mu.Lock()
	defer handlers.Estado.Mu.Unlock()

	evento := models.Event{
		Id:    handlers.Estado.SequenceNumber,
		Value: mensaje,
		Nodo:  nodo,
	}
	handlers.Estado.SequenceNumber++
	handlers.Estado.LastMessage = mensaje
	handlers.Estado.EventLog = append(handlers.Estado.EventLog, evento)
	saveEstado()
}

// Guarda el estado del nodo en un archivo JSON
// Si hay un error al serializar o escribir el archivo, se genera un panic
func saveEstado() {
	path := "/app/nodo.json"

	data, err := json.MarshalIndent(handlers.Estado, "", "  ")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		panic(err)
	}
}
