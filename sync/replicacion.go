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

func RegistrarYReplicarEventos(destinoNodo int) {
	nodoStr := strconv.Itoa(destinoNodo)

	// Agregar eventos al estado local
	agregarEvento("Jugando con la pelota", nodoStr)
	agregarEvento("Se le cayó la pelota", nodoStr)
	agregarEvento("Buscando la pelota", nodoStr)

	// Replicar eventos a los demás nodos
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
				for _, e := range handlers.Estado.EventLog {
					logs.EventLog = append(logs.EventLog, &proto.Event{
						Id:    int32(e.Id),
						Value: e.Value,
						Nodo:  e.Nodo,
					})
				}

				_, err = client.Replicar(ctx, logs)
				if err != nil {
					log.Printf("Error replicando logs a %s: %v", direccion, err)
				} else {
					log.Printf("Logs replicados exitosamente a %s", direccion)
				}
			}(direccion)
		}
	}
}

func agregarEvento(mensaje string, nodo string) {
	handlers.Estado.Mu.Lock()
	defer handlers.Estado.Mu.Unlock()

	handlers.Estado.SequenceNumber++
	evento := models.Event{
		Id:    handlers.Estado.SequenceNumber,
		Value: mensaje,
		Nodo:  nodo,
	}
	handlers.Estado.LastMessage = mensaje
	handlers.Estado.EventLog = append(handlers.Estado.EventLog, evento)
	saveEstado()
	log.Printf(mensaje)
}

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
