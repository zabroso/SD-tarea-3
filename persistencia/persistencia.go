package persistencia

import (
	"SD-Tarea-3/handlers"
	"SD-Tarea-3/proto"
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

// Sincroniza el estado local con el del nodo coordinador en caso de desfasaje
func SincronizarConCoordinador() {
	coordinadorID := handlers.PrimaryNodeID
	direccion, ok := handlers.Nodes[coordinadorID]
	if !ok {
		log.Printf("No se encontró la dirección del coordinador (ID: %d)", coordinadorID)
		return
	}

	conn, err := grpc.Dial(direccion, grpc.WithInsecure())
	if err != nil {
		log.Printf("Error al conectar con el coordinador para verificar secuencia: %v", err)
		return
	}
	defer conn.Close()

	client := proto.NewNodoServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Solicita la comparación del sequence number
	seqReq := &proto.Sequence{SequenceNumber: int32(handlers.Estado.SequenceNumber)}
	resp, err := client.RecuperarSequence(ctx, seqReq)
	if err != nil {
		log.Printf("Error al solicitar secuencia al coordinador: %v", err)
		return
	}

	// Si el nodo está desfasado, solicita recuperación
	if !resp.Desfasado {
		log.Println("Nodo desfasado. Iniciando recuperación de logs...")

		recuperarLogs, err := client.Recuperar(ctx, &proto.Logs{})
		if err != nil {
			log.Printf("Error al recuperar logs del coordinador: %v", err)
			return
		}

		log.Printf("Logs sincronizados exitosamente. Nuevo SequenceNumber: %d", recuperarLogs.SequenceNumber)
	} else {
		log.Println("El nodo está sincronizado con el coordinador.")
	}
}
