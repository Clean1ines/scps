# Dockerfile
FROM golang:1.18 as builder
WORKDIR /app
# Копирование всех исходников проекта в контейнер
COPY . .
# Сборка бинарного файла с именем scps
RUN go build -o scps .
FROM alpine:latest
WORKDIR /app
# Копирование собранного бинарника из контейнера сборки
COPY --from=builder /app/scps .
EXPOSE 8080
# Запуск приложения при старте контейнера
CMD ["./scps"]