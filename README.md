# SD-tarea-3

## Proto command for main
protoc --proto_path=proto proto/*.proto --go_out=.

protoc --proto_path=proto proto/*.proto --go-grpc_out=.

## Proto command for coordinacion
protoc --proto_path=coordinacion/proto coordinacion/proto/*.proto --go_out=coordinacion/gen/

protoc --proto_path=coordinacion/proto coordinacion/proto/*.proto --go-grpc_out=coordinacion/gen/