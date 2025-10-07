# Builder Stage (Usiamo 1.19.1 come richiesto)
FROM golang:1.22 AS builder
WORKDIR /app

# Passo 1: Installiamo i pacchetti di sistema necessari per CGO e SQLite
# dpkg-dev e gcc sono necessari per compilare il driver go-sqlite3.
RUN apt-get update && apt-get install -y --no-install-recommends \
    dpkg-dev \
    gcc \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Passo 6: Compilazione. L'eseguibile sarà /webapi
# Non usiamo CGO_ENABLED=0 perché siamo su Debian (glibc) e dobbiamo supportare SQLite CGO.
RUN go build -ldflags "-s -w" -o /webapi ./cmd/webapi 

# Final Stage (Minimalist runtime basato su Debian Bullseye)
FROM debian:bookworm
EXPOSE 8080

# Passo 8: Installiamo SOLO la libreria di runtime SQLite.
# Questo mantiene l'immagine finale piccola e sicura, non includendo gli strumenti di sviluppo.
RUN apt-get update && apt-get install -y --no-install-recommends \
    libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Passo 10: Copiamo l'eseguibile compilato dal builder
COPY --from=builder /webapi ./webapi 

# Passo 11: Crea il file DB vuoto
RUN touch /app/wasa.db

# Passo 13: Esegue l'eseguibile (porta di ascolto 8080 definita nel docker-compose)
CMD ["./webapi"]
