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

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

// var estado *models.Nodo
var bully *models.Bully

// var nodes map[int]string // Mapa de nodos: ID → dirección IP:puerto
var nodoID int
var primaryNodeID int = -1
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

	log.Printf("Estado cargado: %+v\n", handlers.Estado)

	go iniciarServidorGRPC()

	time.Sleep(2 * time.Second)

	go GetIds()

	time.Sleep(2 * time.Second)

	if handlers.Estado.IsPrimary {
		go func() {
			for {
				log.Println("Nodo coordinador esperando para enviar pelota...")
				<-continuarCiclo
				log.Println("Nodo coordinador. Iniciando ronda de envío aleatorio de pelotas...")

				// Crear lista de nodos destino
				var candidatos []int
				for nodo, direccion := range handlers.Nodes {
					if direccion != handlers.Nodes[nodoID] {
						candidatos = append(candidatos, nodo)
					}
				}

				var destinoNodo int
				if rand.Float64() < 0.5 {
					destinoNodo = candidatos[0]
				} else {
					destinoNodo = candidatos[1]
				}

				destino := handlers.Nodes[destinoNodo]
				log.Printf("Enviando pelota a %d (%s)...\n", destinoNodo, destino)

				ok := enviarPelota(destino, handlers.Estado.ID)
				if !ok {
					log.Println("Fallo al enviar la pelota. Esperando para intentar de nuevo...")
					// Opcional: reintentar luego de un tiempo
					time.AfterFunc(5*time.Second, func() {
						continuarCiclo <- struct{}{}
					})
				}
			}
		}()
		continuarCiclo <- struct{}{}
	} else {
		go func() {
			for {
				log.Println("Esperando señal para verificar HeartBeat...")
				<-continuarCiclo
				destino := handlers.Nodes[primaryNodeID]
				log.Printf("Destino: %v", destino)
				conn, err := grpc.Dial(destino, grpc.WithInsecure())
				if err != nil {

					// Activar algoritmo del maton

					bully.ID = nodoID
					bully.Nodes = handlers.Nodes
					bully.LeaderID = primaryNodeID

					newNodes := coordinacion.StartElection(bully)
					handlers.SetNodes(newNodes)
					continue
				}

				ok := monitoreo.ListenHeartBeat(conn, destino, bully, handlers.Nodes)
				conn.Close()

				if !ok {
					log.Println("Fallo al escuchar latido. Esperando para intentar de nuevo...")
					// Opcional: reintentar luego de un tiempo
					time.AfterFunc(5*time.Second, func() {
						continuarCiclo <- struct{}{}
					})
				} else {

					time.AfterFunc(5*time.Second, func() {
						continuarCiclo <- struct{}{}
					})
				}
			}

		}()
		continuarCiclo <- struct{}{}

	}

	select {}
}

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
	log.Printf("Recibida pelota de %s", req.FromId)

	go func() {
		if !handlers.Estado.IsPrimary {
			ejecutarSimulacion()
		}

		if !handlers.Estado.IsPrimary {
			peerNodeId, err := strconv.Atoi(req.FromId)
			if err != nil {
				log.Printf("Error al convertir FromId a entero: %v", err)
				return
			}
			destino := handlers.Nodes[peerNodeId]
			enviarPelota(destino, handlers.Estado.ID)
		} else {
			continuarCiclo <- struct{}{}
		}
	}()

	return &proto.BallResponse{Message: "Pelota Recibida"}, nil
}

func ejecutarSimulacion() {
	eventos := []string{
		"Jugando con la pelota",
		"Se le cayó la pelota",
		"Buscando la pelota",
		"Devuelve la pelota al coordinador",
	}

	for _, e := range eventos {
		time.Sleep(1 * time.Second)
		agregarEvento(e)
	}
}

func (s *server) Election(ctx context.Context, req *proto.ElectionRequest) (*proto.ElectionResponse, error) {
	log.Printf("Recibida petición de elección de nodo %d", req.SenderId)

	return &proto.ElectionResponse{Ok: true}, nil
}

func (s *server) HeartBeat(ctx context.Context, req *proto.BeatRequest) (*proto.BeatResponse, error) {
	log.Printf("Recibido HeartBeat de %s: %s", req.FromId, req.Message)

	handlers.Estado.LastMessage = time.Now().Format(time.RFC3339)

	return &proto.BeatResponse{FromId: handlers.Estado.ID, Message: "Ok", IsPrimary: handlers.Estado.IsPrimary}, nil
}

// Obtiene las direcciones de los nodos con sus respectivos ids y actualiza la variable `nodes`
func GetIds() {
	for nodo, direccion := range ipMap {
		if direccion != handlers.Nodes[nodoID] {
			log.Printf("Direccion en GetIds de %d: %s", nodoID, direccion)
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
			log.Printf("Enviando HeartBeat a %s desde %s", nodo, handlers.Estado.ID)

			resp, err := client.HeartBeat(ctx, &proto.BeatRequest{FromId: handlers.Estado.ID, Message: "Acknowledged"})
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
				primaryNodeID = peerNodeId
				log.Printf("Nodo %d es el coordinador", primaryNodeID)
			} else {
				log.Printf("Nodo %d no es el coordinador", peerNodeId)
			}
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = client.SendBall(ctx, &proto.BallRequest{FromId: desde})
	if err != nil {
		log.Printf("Error al enviar pelota a %s: %v", destino, err)
		return false
	}

	log.Printf("Pelota enviada a %s", destino)
	return true
}

func agregarEvento(mensaje string) {
	handlers.Estado.Mu.Lock()
	defer handlers.Estado.Mu.Unlock()
	handlers.Estado.SequenceNumber++
	handlers.Estado.LastMessage = mensaje
	handlers.Estado.EventLog = append(handlers.Estado.EventLog, fmt.Sprintf("[#%d] %s", handlers.Estado.SequenceNumber, mensaje))
	saveEstado()
	log.Printf(mensaje)
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
