#!/bin/sh
# start.sh – Скрипт для сборки и запуска проекта в iSH

export TELEGRAM_BOT_TOKEN="7683033592AAGdvZdkMkl3-rjSRp3YHRYIwWEVg06gRXY"
export WEBHOOK_URL="https://youtify-211829086557.us-central1.run.app"
export REDIS_ADDRESS="localhost:6379"
export PORT="8080"
export SPOTIFY_CLIENT_ID="19031702dc8e4866a231f755411d8877"
export SPOTIFY_CLIENT_SECRET="cd45d96a403c4b9e8d651052099d6020"
export SPOTIFY_REDIRECT_URI="https://youtify-211829086557.us-central1.run.app/spotify/callback"
export YOUTUBE_CLIENT_ID="your_youtube_client_id"
export YOUTUBE_CLIENT_SECRET="your_youtube_client_secret"
export YOUTUBE_REDIRECT_URI="https://youtify-211829086557.us-central1.run.app/youtube/callback"
export GOOGLE_CLOUD_PROJECT="youtifyBot"
export ACOUSTID_API_KEY="EMX96S9tia"
export DEFAULT_SPOTIFY_PLAYLIST_ID="your_default_spotify_playlist_id"
export DEFAULT_YOUTUBE_PLAYLIST_ID="your_default_youtube_playlist_id"

echo "Переменные окружения установлены."

if ! pgrep redis-server > /dev/null 2>&1; then
    echo "Запускаем redis-server..."
    redis-server &
    sleep 2
else
    echo "redis-server уже запущен."
fi

echo "Сборка проекта..."
go build -o scps .
if [ $? -ne 0 ]; then
    echo "Ошибка сборки. Проверьте код и зависимости."
    exit 1
fi
echo "Проект успешно собран."

echo "Запуск приложения..."
./scps