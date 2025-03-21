# Используем официальный образ Go для сборки
FROM golang:1.18 AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем файлы go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник для Linux (OS по умолчанию для Cloud Run)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# Второй этап: минимальный образ
FROM alpine:latest

# Копируем бинарник из этапа сборки
COPY --from=builder /app/app /app/app

# Определяем порт, на котором будет слушать приложение
ENV PORT=8080

# Указываем команду для запуска приложения
ENTRYPOINT ["/app/app"]
