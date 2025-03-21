package oauth

import (
	"context"
	"log"
	"net/http"
	"os"

	spotifyOAuth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
	// Для YouTube OAuth можно использовать Google API Client (если требуется)
)

// Глобальные переменные для OAuth-конфигурации
var (
	SpotifyConfig *oauth2.Config
	// YouTubeConfig *oauth2.Config // При необходимости
)

// InitOAuthConfig инициализирует конфигурации OAuth для Spotify и YouTube
func InitOAuthConfig() {
	// Инициализация OAuth для Spotify
	SpotifyConfig = &oauth2.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("SPOTIFY_REDIRECT_URI"),
		Scopes:       []string{spotifyOAuth.ScopeUserLibraryRead, spotifyOAuth.ScopePlaylistModifyPrivate, spotifyOAuth.ScopePlaylistModifyPublic},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
	}
	// Аналогично можно инициализировать конфигурацию для YouTube, если требуется
	log.Println("OAuth конфигурация для Spotify и YouTube инициализирована")
}

// SpotifyOAuthHandler обрабатывает редирект для Spotify OAuth
func SpotifyOAuthHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем код авторизации из запроса
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Отсутствует код авторизации", http.StatusBadRequest)
		return
	}

	// Обмениваем код на токен
	token, err := SpotifyConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Ошибка обмена кода на токен: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Здесь можно сохранить токен в базе данных, привязав к сессии пользователя
	// Например, db.UpdateUserSession(userID, "spotify_token", token.AccessToken)

	// Отправляем подтверждение пользователю или редиректим на нужную страницу
	w.Write([]byte("Spotify OAuth успешно завершён! Теперь бот может работать с вашим аккаунтом Spotify."))
}
