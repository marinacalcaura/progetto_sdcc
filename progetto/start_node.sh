#!/bin/bash

x=${1:-0}  # If no argument is provided, set x to 0 by default

# Read the value of 'm' from config.json
m=$(cat config.json | jq -r '.num_nodi')

# Read the value of 'm' from config.json
BIT=$(cat config.json | jq -r '.num_bit')



# Calculate the port based on '8005 + m + x'
node_number=$((m + x + 1 ))

port=$((8000 + node_number))

# Build the Docker image
docker build -t "sdcc2_node${node_number}" --build-arg BIT=$BIT -f node/Dockerfile .

# Run the Docker container with the calculated port (--rm permette il riavvio del container senza usare i flag).
docker run --rm -p $port:$port --network=sdcc2_chord-network -e NODE_PORT=$port -e BIT=$BIT "sdcc2_node${node_number}"
