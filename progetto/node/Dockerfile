# Usa un'immagine di base Go
FROM golang:1.19

# Imposta la directory di lavoro all'interno del contenitore
WORKDIR /app

# Copia i file del tuo componente nell'immagine Docker
COPY ./node/node.go /app/node.go

# Copia i file go.mod e go.sum nella directory di lavoro
COPY go.mod ./

# Installa le dipendenze usando go mod
RUN go mod download

# Compila l'applicazione (puoi aggiungere ulteriori comandi per gestire le dipendenze qui)
RUN go build -o node

# Comando di avvio dell'applicazione
ENTRYPOINT [ "./node"]