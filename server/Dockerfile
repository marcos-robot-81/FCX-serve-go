# Estágio de Compilação (Build)
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copia os arquivos de dependência
COPY go.mod go.sum ./
RUN go mod download

# Copia o código fonte
COPY . .

# Compila o binário de forma estática (funciona em qualquer Linux)
RUN CGO_ENABLED=0 go build -o server main.go

# Estágio Final (Imagem leve para rodar)
FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/server .

# Copia a pasta de templates para a imagem final
COPY templates ./templates

EXPOSE 8080
CMD ["./server"]