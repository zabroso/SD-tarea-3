package coordinacion

import (
	"SD-Tarea-3/handlers"
	"SD-Tarea-3/models"
	"SD-Tarea-3/proto"
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
)

func StartElection(b *models.Bully) {
	log.Println("Se ejecuta algoritmo del Matón")
	receivedOK := false

	for peerID, address := range b.Nodes {
		if peerID <= b.ID {
			delete(handlers.Nodes, peerID)
			continue
		}

		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			fmt.Printf("Nodo %d no respondió (timeout): %s\n", peerID, address)
			continue
		}

		client := proto.NewNodoServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := client.Election(ctx, &proto.ElectionRequest{SenderId: int32(b.ID)})
		conn.Close()

		if err == nil && resp.Ok {
			fmt.Printf("Nodo %d respondió OK\n", peerID)
			receivedOK = true
		} else {
			delete(handlers.Nodes, peerID)
		}
	}

	if !receivedOK {
		handlers.NewPrimary()
		AnnounceCoordinator(b)
	} else {
		fmt.Println("Esperando que otro nodo anuncie al nuevo coordinador...")
		time.Sleep(5 * time.Second)
	}

}

func AnnounceCoordinator(b *models.Bully) {
	b.LeaderID = b.ID
	fmt.Printf("Nodo %d se proclama líder\n", b.ID)

	for peerID, address := range b.Nodes {
		if peerID == b.ID {
			continue
		}

		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			fmt.Printf("Nodo %d no respondió (timeout): %s\n", peerID, address)
			continue
		}

		client := proto.NewNodoServiceClient(conn)
		client.Coordinator(context.Background(), &proto.CoordinatorMessage{CoordinatorId: int32(b.ID)})
		conn.Close()
	}
}
