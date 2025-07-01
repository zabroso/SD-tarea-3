package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"SD-Tarea-3/coordinacion"
	"SD-Tarea-3/handlers"
	"SD-Tarea-3/models"
	"SD-Tarea-3/monitoreo"
	"SD-Tarea-3/proto"
	replicacion "SD-Tarea-3/sync"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

var bully *models.Bully
var nodoID int

var ipMap map[string]string
var continuarCiclo = make(chan struct{})

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error cargando .env")
	}
	var err error
	nodoID, err = strconv.Atoi(os.Getenv("ID_NODO"))
	if err != nil {
		log.Fatalf("ID_NODO inválido: %v", err)
	}
	handlers.Nodes = make(map[int]string)

	handlers.Nodes[nodoID] = os.Getenv("IP_NODO") + ":" + os.Getenv("PORT_NODO")
	bully = &models.Bully{}
	ipMap = map[string]string{
		"nodo0": os.Getenv("IP_VM1") + ":" + os.Getenv("PORT_VM1"),
		"nodo1": os.Getenv("IP_VM2") + ":" + os.Getenv("PORT_VM2"),
		"nodo2": os.Getenv("IP_VM3") + ":" + os.Getenv("PORT_VM3"),
	}

	handlers.Estado = cargarEstado("nodo.json")
	isPrimaryStr := os.Getenv("IS_PRIMARY")
	handlers.Estado.IsPrimary = isPrimaryStr == "true"
	handlers.Estado.ID = os.Getenv("ID_NODO")
	handlers.Estado.Port, err = strconv.Atoi(os.Getenv("PORT_NODO"))
	if err != nil {
		log.Fatalf("PORT_NODO inválido: %v\n", err)
	}

	log.Printf("Estado cargado: ID=%s, SequenceNumber=%d, EventLog=%+v\n",
		handlers.Estado.ID,
		handlers.Estado.SequenceNumber,
		handlers.Estado.EventLog,
	)

	go iniciarServidorGRPC()

	time.Sleep(2 * time.Second)

	go GetIds()

	time.Sleep(2 * time.Second)

	go func() {
		for {
			if handlers.Estado.IsPrimary {
				funcionalidadPrimario()
			} else {
				funcionalidadSecundario()
			}
		}
	}()

	continuarCiclo <- struct{}{}
	select {}
}

func funcionalidadPrimario() {
	<-continuarCiclo
	log.Println("Nodo coordinador. Iniciando ronda de envío aleatorio de pelotas...")

	var candidatos []int
	for nodo, direccion := range handlers.Nodes {
		if direccion != handlers.Nodes[nodoID] {
			candidatos = append(candidatos, nodo)
		}
	}

	if len(candidatos) == 0 {
		log.Println("No hay nodos disponibles para enviar la pelota.")
		time.AfterFunc(5*time.Second, func() {
			continuarCiclo <- struct{}{}
		})
		return
	}

	destinoNodo := candidatos[rand.Intn(len(candidatos))]
	destino := handlers.Nodes[destinoNodo]
	log.Printf("Enviando pelota a %d (%s)...\n", destinoNodo, destino)

	ok := enviarPelota(destino, handlers.Estado.ID)

	if ok {
		replicacion.RegistrarYReplicarEventos(destinoNodo)
	} else {
		log.Println("Fallo al enviar la pelota. Esperando para intentar de nuevo...")
	}

	time.AfterFunc(5*time.Second, func() {
		continuarCiclo <- struct{}{}
	})
}

func funcionalidadSecundario() {
	<-continuarCiclo
	destino := handlers.Nodes[handlers.PrimaryNodeID]
	conn, err := grpc.Dial(destino, grpc.WithInsecure())
	if err != nil {
		bully.ID = nodoID
		bully.Nodes = handlers.Nodes
		bully.LeaderID = handlers.PrimaryNodeID

		coordinacion.StartElection(bully)

		return
	}

	monitoreo.ListenHeartBeat(conn, destino, bully, handlers.Nodes)
	conn.Close()

	time.AfterFunc(5*time.Second, func() {
		continuarCiclo <- struct{}{}
	})
}

// Inicia el servidor gRPC y registra el servicio NodoService
func iniciarServidorGRPC() {
	ip := os.Getenv("IP_NODO")
	port := os.Getenv("PORT_NODO")
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		log.Fatalf("Error al escuchar: %v", err)
	}

	s := grpc.NewServer()
	proto.RegisterNodoServiceServer(s, &server{})
	log.Printf("Nodo %s escuchando en %s:%s", handlers.Estado.ID, ip, port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Fallo al servir: %v", err)
	}
}

type server struct {
	proto.UnimplementedNodoServiceServer
}

func (s *server) SendBall(ctx context.Context, req *proto.BallRequest) (*proto.BallResponse, error) {
	log.Printf("Recibida pelota de %s", req.GetFromId())

	logs := req.GetLogs()

	if !handlers.Estado.IsPrimary {
		for _, event := range logs.GetEventLog() {
			log.Printf("%s\n", event.GetValue())
			select {
			case <-ctx.Done():
				log.Printf("Contexto cancelado, abortando simulación")
				return nil, ctx.Err()
			case <-time.After(4 * time.Second):
			}
		}
	}

	return &proto.BallResponse{Emulado: true}, nil
}

func (s *server) Replicar(ctx context.Context, req *proto.Logs) (*proto.ReplicarResponse, error) {
	for _, e := range req.GetEventLog() {
		agregarEvento(e.Value, e.Nodo)
	}

	log.Printf("Logs replicados y guardados correctamente. SequenceNumber: %d", handlers.Estado.SequenceNumber)

	return &proto.ReplicarResponse{SequenceNumber: int32(handlers.Estado.SequenceNumber)}, nil
}

// Informa a los nodos del nuevo coordinador
func (s *server) Coordinator(ctx context.Context, req *proto.CoordinatorMessage) (*proto.Empty, error) {
	log.Printf("Recibida petición de coordinador de nodo %d", req.CoordinatorId)

	handlers.PrimaryNodeID = int(req.CoordinatorId)

	return &proto.Empty{}, nil
}

// Maneja la petición de elección de nodo
func (s *server) Election(ctx context.Context, req *proto.ElectionRequest) (*proto.ElectionResponse, error) {
	log.Printf("Recibida petición de elección de nodo %d", req.SenderId)

	return &proto.ElectionResponse{Ok: true}, nil
}

// Se encarga de verificar que el nodo sigue activo. También es utilizado para obtener los ids de los nodos al inicio
func (s *server) HeartBeat(ctx context.Context, req *proto.BeatRequest) (*proto.BeatResponse, error) {
	id, err := strconv.Atoi(req.FromId)
	if err != nil {
		log.Printf("Error al convertir FromId a entero: %v", err)
		return nil, err
	}

	if _, exists := handlers.Nodes[id]; !exists {
		log.Printf("Reintegrando nodo %d con dirección %s", id, req.Ip)
		handlers.Nodes[id] = req.Ip
	}
	handlers.Estado.LastMessage = time.Now().Format(time.RFC3339)

	return &proto.BeatResponse{FromId: handlers.Estado.ID, Message: "Ok", IsPrimary: handlers.Estado.IsPrimary}, nil
}

// Obtiene las direcciones de los nodos con sus respectivos ids y actualiza la variable `nodes`.
// Ayuda a reintegrar nodos cuando se reinician
func GetIds() {
	for nodo, direccion := range ipMap {
		if direccion != handlers.Nodes[nodoID] {
			conn, err := grpc.Dial(direccion, grpc.WithInsecure())
			if err != nil {
				log.Printf("Error direccion %s", direccion)
				log.Printf("Error al conectar con %s: %v", nodo, err)
				continue
			}

			defer conn.Close()

			client := proto.NewNodoServiceClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			resp, err := client.HeartBeat(ctx, &proto.BeatRequest{FromId: handlers.Estado.ID, Message: "Acknowledged", Ip: handlers.Nodes[nodoID]})
			if err != nil {
				log.Printf("Error al enviar HeartBeat a %s: %v", nodo, err)
				continue
			}
			peerNodeId, err := strconv.Atoi(resp.FromId)
			if err != nil {
				log.Printf("Error al convertir FromId a entero: %v", err)
				continue
			}
			handlers.Nodes[peerNodeId] = direccion
			if resp.IsPrimary {
				log.Printf("Nodo %d es el coordinador actual", peerNodeId)
				handlers.PrimaryNodeID = peerNodeId
			}
		}

	}

	if handlers.PrimaryNodeID == -1 {
		var maxID int
		for id := range handlers.Nodes {
			if id > maxID {
				maxID = id
			}
		}
		handlers.PrimaryNodeID = maxID

		if handlers.PrimaryNodeID == nodoID {
			handlers.Estado.IsPrimary = true
		}
	}
}

func enviarPelota(destino string, desde string) bool {
	conn, err := grpc.Dial(destino, grpc.WithInsecure())
	if err != nil {
		log.Printf("Error al conectar con %s: %v", destino, err)
		return false
	}
	defer conn.Close()

	client := proto.NewNodoServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	eventBaseID := handlers.Estado.SequenceNumber
	eventos := []*proto.Event{
		{Id: int32(eventBaseID), Value: "Jugando con la pelota"},
		{Id: int32(eventBaseID + 1), Value: "Se le cayó la pelota"},
		{Id: int32(eventBaseID + 2), Value: "Buscando la pelota"},
	}
	nuevoSequence := eventBaseID + 3

	logs := &proto.Logs{
		SequenceNumber: int32(nuevoSequence),
		EventLog:       eventos,
	}

	// Actualizar el estado local con nuevo sequenceNumber
	// handlers.Estado.SequenceNumber = nuevoSequence

	// Enviar la pelota
	resp, err := client.SendBall(ctx, &proto.BallRequest{
		FromId: desde,
		Logs:   logs,
	})

	if err != nil {
		log.Printf("Error: servidor no respondió o se cayó: %v", err)
		return false
	}

	log.Println("Respuesta recibida:", resp.GetEmulado())
	log.Printf("Pelota enviada a %s", destino)
	return true
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
}

func cargarEstado(path string) *models.Nodo {
	file, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var n models.Nodo
	if err := json.Unmarshal(file, &n); err != nil {
		panic(err)
	}
	return &n
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
