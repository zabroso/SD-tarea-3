package monitoreo

import (
	"SD-Tarea-3/models"
	"SD-Tarea-3/proto"
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

func HeartBeat(conn *grpc.ClientConn, estado *models.Nodo, destino string) {
	client := proto.NewNodoHeartBeatClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.HeartBeat(ctx, &proto.BeatRequest{FromId: estado.ID})
	if err != nil {
		log.Printf("Error al enviar ack a %s: %v", destino, err)
	}

	log.Printf("Ack enviado a %s", destino)
}
