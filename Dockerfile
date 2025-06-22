# Usa la imagen oficial de Go
FROM golang:1.24

# Establece el directorio de trabajo dentro del contenedor
WORKDIR /app

# Copia go.mod y go.sum primero (para aprovechar cache)
COPY go.mod go.sum ./

# Descarga dependencias
RUN go mod download

# Copia todo el código
COPY . .

# Compila el programa
RUN go build -o nodo main.go

# Comando por defecto al ejecutar el contenedor
CMD ["./nodo"]
