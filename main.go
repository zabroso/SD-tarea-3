// main.go
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
	"sync"
	"time"

	"SD-Tarea-3/proto"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

type Nodo struct {
	ID             string   `json:"id"`
	IsPrimary      bool     `json:"is_primary"`
	LastMessage    string   `json:"last_message"`
	SequenceNumber int      `json:"sequence_number"`
	EventLog       []string `json:"event_log"`
	Port           int      `json:"port"`
	mu             sync.Mutex
}

var estado *Nodo
var nodoID string
var ipMap map[string]string
var portMap map[string]string
var continuarCiclo = make(chan struct{})

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error cargando .env")
	}

	nodoIDInt, err := strconv.Atoi(os.Getenv("ID_NODO"))
	if err != nil {
		log.Fatalf("ID_NODO inválido: %v", err)
	}

	nodoID = fmt.Sprintf("nodo%d", nodoIDInt)

	ipMap = map[string]string{
		"nodo0": os.Getenv("IP_VM1"),
		"nodo1": os.Getenv("IP_VM2"),
		"nodo2": os.Getenv("IP_VM3"),
	}
	portMap = map[string]string{
		"nodo0": os.Getenv("PORT_VM1"),
		"nodo1": os.Getenv("PORT_VM2"),
		"nodo2": os.Getenv("PORT_VM3"),
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

	if estado.IsPrimary {
		go func() {
			for {
				fmt.Println("Nodo coordinador esperando para enviar pelota...")
				<-continuarCiclo
				fmt.Println("Nodo coordinador. Iniciando ronda de envío aleatorio de pelotas...")

				// Crear lista de nodos destino
				var candidatos []string
				for nodo, puerto := range portMap {
					puertoInt, err := strconv.Atoi(puerto)
					if err != nil {
						continue
					}
					if puertoInt != estado.Port {
						candidatos = append(candidatos, nodo)
					}
				}

				var destinoNodo string
				if rand.Float64() < 0.5 {
					destinoNodo = candidatos[0]
				} else {
					destinoNodo = candidatos[1]
				}

				destino := fmt.Sprintf("%s:%s", ipMap[destinoNodo], portMap[destinoNodo])
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

	go func() {
		if !estado.IsPrimary {
			ejecutarSimulacion()
		}

		if !estado.IsPrimary {
			destino := fmt.Sprintf("%s:%s", ipMap["nodo2"], portMap["nodo2"])
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
	estado.mu.Lock()
	defer estado.mu.Unlock()
	estado.SequenceNumber++
	estado.LastMessage = mensaje
	estado.EventLog = append(estado.EventLog, fmt.Sprintf("[#%d] %s", estado.SequenceNumber, mensaje))
	saveEstado(fmt.Sprintf("/app/nodo_%s.json", estado.ID))
	fmt.Println(mensaje)
}

func cargarEstado(path string) *Nodo {
	file, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var n Nodo
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
