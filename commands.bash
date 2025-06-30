if [ -z "$1" ] || [ -z "$2" ]; then
  echo "Uso: commands <kill/start> <id>"
  exit 1
fi

COMMAND="$1"
ID="$2"
CONTAINER=""

case "$ID" in
  0) CONTAINER="nodo0" ;;
  1) CONTAINER="nodo1" ;;
  2) CONTAINER="nodo2" ;;
  *) echo "ID inválido. Usa 0, 1 o 2." && exit 1 ;;
esac

case "$COMMAND" in
  kill) DOCKER_COMMAND="stop" ;;
  start) DOCKER_COMMAND="start" ;;
  *) echo "Comando inválido. Usa 'kill' o 'start'." && exit 1 ;;
esac

echo "Ejecutando '$DOCKER_COMMAND' en el contenedor $CONTAINER..."
docker "$DOCKER_COMMAND" "$CONTAINER"
