package monitoreo

import (
	"SD-Tarea-3/coordinacion"
	"SD-Tarea-3/models"
	"SD-Tarea-3/proto"
	"context"
	"log"
	"strconv"
	"time"

	"google.golang.org/grpc"
)

var estado *models.Nodo

func ListenHeartBeat(conn *grpc.ClientConn, destino string, bully *models.Bully, nodes map[int]string) {
	client := proto.NewNodoServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.HeartBeat(ctx, &proto.BeatRequest{FromId: estado.ID, Message: "Acknowledged"})
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Timeout al enviar ack a %s", destino)
			bully.ID, err = strconv.Atoi(estado.ID)
			if err != nil {
				log.Printf("Error al convertir ID de nodo a entero: %v", err)
			}
			bully.Nodes = nodes
			coordinacion.StartElection(bully)

		} else {
			log.Printf("Error al enviar ack a %s: %v", destino, err)
		}
	}

	log.Printf("Ack enviado a %s", destino)
}

func SetEstado(estadoNodo *models.Nodo) {
	estado = estadoNodo
	log.Printf("Estado del nodo actualizado: ID=%s", estado.ID)
}
