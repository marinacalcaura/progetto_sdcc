#!/bin/bash

# Verifica se Docker Compose è installato
if ! command -v docker-compose &> /dev/null; then
    echo "Docker Compose non è installato. Installalo prima di continuare."
    exit 1
fi

# Verifica se il file docker-compose.yml esiste
if [ ! -f docker-compose.yml ]; then
    echo "Il file docker-compose.yml non è presente nella directory corrente."
    exit 1
fi

# Avvia l'applicazione Docker Compose
docker-compose up -d

# Controlla lo stato dei contenitori avviati
if docker-compose ps | grep "Up" >/dev/null; then
    echo "L'applicazione è stata avviata con successo."
else
    echo "Si è verificato un errore durante l'avvio dell'applicazione."
fi
