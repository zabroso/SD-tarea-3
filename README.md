# Tarea 3 Sistemas Distribuidos

## Integrantes

| Nombre           | Rol           |
|------------------|---------------|
| Benjamín Campos  | 202073128-4   |
| Pablo Marambio   | 202073108-k   |

## Instrucciones de montaje del sistema

* Nuestra tarea se adaptó para poder ser ejecutada fuera de las maquinas virtuales debido a que no se nos pudieron entregar. Para ello realizamos la tarea con Docker, con tal de simular 3 maquinas distintas.

* Los requisitos serían:

  * Docker (Idealmente Docker Desktop para mayor comodidad: https://docs.docker.com/desktop/ , sino el Docker Engine: https://docs.docker.com/engine/install/)
  * WSL

Para poder montar el sistema se debe ejecutar el siguiente comando:

    docker-compose up -d --build

* Cuando terminen de crearse los contenedores, estarán los 3 nodos arriba

## Instrucciones de uso y funcionamiento

* Los nodos una vez iniciados, comienzan inmediatamente la simulación de eventos, donde en nuestro caso simulamos que el nodo primario va enviando una "pelota" a algun nodo secundario, el nodo secundario hará algo con la "pelota" y cuando termina de hacer ese algo, le avisa al primario que ha terminado.

* Los comandos para matar o iniciar los nodos son los siguientes (Asegurarse de que si estan en Visual Studio Code, en la barrita negra inferior debe estar LF y no CRLF al cuando se esté en el commands.bash, sino no funcionará):

    * Comando para iniciar un nodo nuevamente: 

            bash commands.bash start [id_nodo]
    
    * Comando para iniciar un nodo nuevamente:
  
            bash commands.bash kill [id_nodo]

* Idealmente dar un lapso de tiempo para que trabajen los nodos antes de dar muerte o iniciar un nodo. Por ejemplo, no iniciar y matar un nodo al mismo tiempo.
* Hay una espera de al menos 15 segundos por evento (Eso incluye a veces el retomar las funciones del primario para un nodo que recién tomó el puesto, a veces demora)
* A veces hay fallas o errores, pero no necesariamente se ha muerto el sistema, es solo el reconocimiento de estos.