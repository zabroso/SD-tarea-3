package main

import (
	"log"
	"net"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func getEnv(key string) string {

	// load .env file
	err := godotenv.Load("./.env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return os.Getenv(key)
}

func main() {
	listener, err := net.Listen("tcp", getEnv("IP_VM1")+":"+getEnv("PORT_VM1"))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	// Register your gRPC services here
	log.Println("Server is running on localhost:50051")
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
