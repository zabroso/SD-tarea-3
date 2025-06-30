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

func StartElection(b *models.Bully) map[int]string {
	log.Println("Bully algorithm executed")
	receivedOK := false

	for peerID, address := range b.Nodes {
		if peerID <= b.ID {
			continue
		}

		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			fmt.Printf("Nodo %d no respondió (timeout): %s\n", peerID, address)
			delete(b.Nodes, peerID)
			continue
		}

		client := proto.NewNodoServiceClient(conn)
		resp, err := client.Election(context.Background(), &proto.ElectionRequest{SenderId: int32(b.ID)})
		conn.Close()

		if err == nil && resp.Ok {
			fmt.Printf("Nodo %d respondió OK\n", peerID)
			receivedOK = true
		}
	}

	if !receivedOK {
		AnnounceCoordinator(b)
	} else {
		fmt.Println("Esperando que otro nodo anuncie al nuevo líder...")
		time.Sleep(5 * time.Second)
	}

	return b.Nodes

}

func AnnounceCoordinator(b *models.Bully) {
	b.LeaderID = b.ID
	handlers.NewPrimary()
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
