package monitoreo

import (
	"SD-Tarea-3/coordinacion"
	"SD-Tarea-3/handlers"
	"SD-Tarea-3/models"
	"SD-Tarea-3/proto"
	"context"
	"log"
	"strconv"
	"time"

	"google.golang.org/grpc"
)

func ListenHeartBeat(conn *grpc.ClientConn, destino string, bully *models.Bully, nodes map[int]string) bool {
	client := proto.NewNodoServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.HeartBeat(ctx, &proto.BeatRequest{FromId: handlers.Estado.ID, Message: "Acknowledged"})
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Timeout al enviar ack a %s", destino)
			bully.ID, err = strconv.Atoi(handlers.Estado.ID)
			if err != nil {
				log.Printf("Error al convertir ID de nodo a entero: %v", err)
			}
			bully.Nodes = nodes
			coordinacion.StartElection(bully)
			return false

		} else {
			log.Printf("Error al enviar ack a %s: %v", destino, err)
			return false
		}
	}

	log.Printf("Ack enviado a %s", destino)
	return true
}
