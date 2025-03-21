package main

import (
	"log"
	"net/http"

	"./oauth"
)

// FirebaseFunctionHandler – HTTP функция для деплоя на Firebase Cloud Functions
// Она может обрабатывать разные пути: например, /spotifyOAuth для OAuth редиректа
func FirebaseFunctionHandler(w http.ResponseWriter, r *http.Request) {
	// Маршрутизатор по пути запроса
	switch r.URL.Path {
	case "/spotifyOAuth":
		oauth.SpotifyOAuthHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

func main() {
	// Для локального тестирования можно запустить HTTP сервер
	http.HandleFunc("/", FirebaseFunctionHandler)
	log.Println("Firebase HTTP функция запущена на порту 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
