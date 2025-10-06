# BUILD STAGE: usa un'immagine Go completa per la compilazione
FROM golang:1.22 AS builder

# Imposta la directory di lavoro all'interno del container
WORKDIR /app

# Copia i file di dipendenza e scaricali (ottimizzazione della cache)
COPY go.mod go.sum ./
RUN go mod download

# Copia il resto del codice sorgente
COPY . .

# Compila l'applicazione. L'eseguibile sarà chiamato 'webapi'
RUN go build -o /webapi ./cmd/webapi

# -------------------------------------------------------------------
# RUN STAGE: usa un'immagine Alpine più leggera per l'esecuzione
FROM alpine:latest

# Installa SQLite, necessario per il database
RUN apk --no-cache add sqlite

# Imposta la directory di lavoro
WORKDIR /app

# Copia l'eseguibile compilato dallo stage precedente
COPY --from=builder /webapi /app/webapi

# Crea un database vuoto (nome tipico nei progetti WASA)
RUN touch /app/wasa.db

# Espone la porta (assumendo che il tuo server ascolti sulla porta 8080)
EXPOSE 8080

# Comando per eseguire l'applicazione
CMD ["./webapi"]