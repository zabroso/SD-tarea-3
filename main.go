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
	"SD-Tarea-3/models"
	"SD-Tarea-3/monitoreo"
	"SD-Tarea-3/proto"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var estado *models.Nodo
var bully *models.Bully
var nodes map[int]string // Mapa de nodos: ID → dirección IP:puerto
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
	nodes = make(map[int]string)

	nodes[nodoID] = os.Getenv("IP_NODO") + ":" + os.Getenv("PORT_NODO")
	bully = &models.Bully{}
	ipMap = map[string]string{
		"nodo0": os.Getenv("IP_VM1") + ":" + os.Getenv("PORT_VM1"),
		"nodo1": os.Getenv("IP_VM2") + ":" + os.Getenv("PORT_VM2"),
		"nodo2": os.Getenv("IP_VM3") + ":" + os.Getenv("PORT_VM3"),
	}

	rutaJson := fmt.Sprintf("nodo_%s.json", os.Getenv("ID_NODO"))
	estado = cargarEstado(rutaJson)
	isPrimaryStr := os.Getenv("IS_PRIMARY")
	estado.IsPrimary = isPrimaryStr == "true"
	estado.ID = os.Getenv("ID_NODO")
	estado.Port, err = strconv.Atoi(os.Getenv("PORT_NODO"))
	if err != nil {
		log.Fatalf("PORT_NODO inválido: %v\n", err)
	}

	fmt.Printf("Estado cargado: %+v\n", estado)

	go iniciarServidorGRPC()

	time.Sleep(2 * time.Second)

	go GetIds()

	time.Sleep(2 * time.Second)

	if estado.IsPrimary {
		go func() {
			for {
				fmt.Println("Nodo coordinador esperando para enviar pelota...")
				<-continuarCiclo
				fmt.Println("Nodo coordinador. Iniciando ronda de envío aleatorio de pelotas...")

				// Crear lista de nodos destino
				var candidatos []string
				for nodo, direccion := range ipMap {
					if err != nil {
						continue
					}
					if direccion != nodes[nodoID] {
						candidatos = append(candidatos, nodo)
					}
				}

				var destinoNodo string
				if rand.Float64() < 0.5 {
					destinoNodo = candidatos[0]
				} else {
					destinoNodo = candidatos[1]
				}

				destino := ipMap[destinoNodo]
				fmt.Printf("Enviando pelota a %s (%s)...\n", destinoNodo, destino)

				ok := enviarPelota(destino, estado.ID)
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
				<-continuarCiclo
				destino := nodes[primaryNodeID]
				log.Printf("Destino: %v", destino)
				conn, err := grpc.Dial(destino, grpc.WithInsecure())
				if err != nil {

					// Activar algoritmo del maton

					bully.ID = nodoID
					bully.Nodes = nodes
					bully.LeaderID = primaryNodeID

					coordinacion.StartElection(bully)
				}
				conn.Close()
				monitoreo.SetEstado(estado)
				monitoreo.ListenHeartBeat(conn, destino, bully, nodes)
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
	log.Printf("Nodo %s escuchando en %s:%s", estado.ID, ip, port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Fallo al servir: %v", err)
	}
}

type server struct {
	proto.UnimplementedNodoServiceServer
}

func (s *server) SendBall(ctx context.Context, req *proto.BallRequest) (*proto.BallResponse, error) {
	log.Printf("Recibida pelota de %s", req.FromId)
	log.Printf("Nodo %s, %s iniciado. Esperando conexiones...", estado.ID, nodes[nodoID])

	go func() {
		if !estado.IsPrimary {
			ejecutarSimulacion()
		}

		if !estado.IsPrimary {
			destino := ipMap["nodo2"]
			enviarPelota(destino, estado.ID)
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

	estado.LastMessage = time.Now().Format(time.RFC3339)

	return &proto.BeatResponse{FromId: estado.ID, Message: "Ok", IsPrimary: estado.IsPrimary}, nil
}

func GetIds() {
	for nodo, direccion := range ipMap {
		if direccion != nodes[nodoID] {
			log.Printf("Direccion en GetIds de %d: %s", nodoID, direccion)
			conn, err := grpc.NewClient(direccion, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Error direccion %s", direccion)
				log.Printf("Error al conectar con %s: %v", nodo, err)
				continue
			}

			defer conn.Close()

			client := proto.NewNodoServiceClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			log.Printf("Enviando HeartBeat a %s desde %s", nodo, estado.ID)
			resp, err := client.HeartBeat(ctx, &proto.BeatRequest{FromId: estado.ID, Message: "Acknowledged"})
			if err != nil {
				log.Printf("Error al enviar HeartBeat a %s: %v", nodo, err)
				cancel()
				conn.Close()
				continue
			}
			peerNodeId, err := strconv.Atoi(resp.FromId)
			if err != nil {
				log.Printf("Error al convertir FromId a entero: %v", err)
				cancel()
				conn.Close()
				continue
			}
			nodes[peerNodeId] = direccion
			if resp.IsPrimary {
				primaryNodeID = peerNodeId
				log.Printf("Nodo %d es el coordinador", primaryNodeID)
			} else {
				log.Printf("Nodo %d no es el coordinador", peerNodeId)
			}
			defer cancel()
			conn.Close()
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
	estado.Mu.Lock()
	defer estado.Mu.Unlock()
	estado.SequenceNumber++
	estado.LastMessage = mensaje
	estado.EventLog = append(estado.EventLog, fmt.Sprintf("[#%d] %s", estado.SequenceNumber, mensaje))
	saveEstado(fmt.Sprintf("/app/nodo_%s.json", estado.ID))
	fmt.Println(mensaje)
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

func saveEstado(path string) {
	data, err := json.MarshalIndent(estado, "", "  ")
	if err != nil {
		panic(err)
	}
	_ = os.WriteFile(path, data, 0644)
}
