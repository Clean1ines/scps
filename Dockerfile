# Dockerfile
FROM golang:1.20-alpine as builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -o bot .

FROM alpine:latest
# Устанавливаем необходимые утилиты: ca-certificates и fpcalc (Chromaprint)
RUN apk --no-cache add ca-certificates fpcalc
WORKDIR /root/
COPY --from=builder /app/bot .
EXPOSE 8080
CMD ["./bot"]