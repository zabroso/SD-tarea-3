package coordinacion

import (
	"SD-Tarea-3/coordinacion/gen/proto"
	"SD-Tarea-3/models"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func getEnv(key string) string {

	// load .env file
	err := godotenv.Load("../.env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return os.Getenv(key)
}

func (b *models.Bully) startElection() {
	// This function would contain the logic for the bully algorithm
	// For now, we will just log that the function was called
	log.Println("Bully algorithm executed")
	receivedOK := false

	for peerID, address := range b.Peers {
		if peerID <= b.ID {
			continue
		}

		conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Printf("Nodo %d no respondió (timeout): %s\n", peerID, address)
			continue
		}

		client := proto.NewNodeServiceClient(conn)
		resp, err := client.Election(context.Background(), &proto.ElectionRequest{SenderId: int32(b.ID)})
		conn.Close()

		if err == nil && resp.Ok {
			fmt.Printf("Nodo %d respondió OK\n", peerID)
			receivedOK = true
		}
	}

	if !receivedOK {
		b.announceCoordinator()
	} else {
		// Espera al coordinador
		fmt.Println("⏳ Esperando que otro nodo anuncie al nuevo líder...")
		time.Sleep(5 * time.Second) // Tiempo de espera arbitrario
		// (Alternativamente, puedes esperar a que llegue un mensaje gRPC)
	}

}

func (b *Bully) announceCoordinator() {
	b.IsLeader = true
	b.LeaderID = b.ID
	fmt.Printf("👑 Nodo %d se proclama líder\n", b.ID)

	for peerID, address := range b.Peers {
		if peerID == b.ID {
			continue
		}

		conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(1*time.Second))
		if err != nil {
			continue
		}

		client := proto.NewNodeServiceClient(conn)
		client.Coordinator(context.Background(), &proto.CoordinatorMessage{CoordinatorId: int32(b.ID)})
		conn.Close()
	}
}
