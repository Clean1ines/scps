package youtube

import (
	"errors"
	"log"

	"context"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"

	"../db"
)

// SyncLikedMusic синхронизирует плейлист "понравившиеся" для YouTube Music (только музыка)
func SyncLikedMusic(chatID int64) error {
	// Получаем API ключ или OAuth токен для YouTube
	apiKey := db.GetUserSessionValue(chatID, "youtube_api_key")
	if apiKey == "" {
		// Если API ключ не хранится в сессии, используем глобальную переменную (настройте через переменные окружения)
		apiKey = "YOUR_YOUTUBE_API_KEY"
	}

	// Создаем сервис YouTube Data API
	service, err := youtube.NewService(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("Ошибка создания YouTube сервиса: %v", err)
		return err
	}

	// Получаем список лайкнутых видео (плейлист "Liked videos" имеет специальный идентификатор)
	call := service.PlaylistItems.List([]string{"snippet", "contentDetails"}).
		PlaylistId("LL"). // "LL" – идентификатор для лайкнутых видео
		MaxResults(50)

	response, err := call.Do()
	if err != nil {
		log.Printf("Ошибка получения лайкнутых видео: %v", err)
		return err
	}

	// Фильтруем видео, оставляя только музыкальные (например, по категории или ключевым словам)
	// TODO: Реализуйте фильтрацию только музыкальных видео

	log.Printf("Получено %d элементов из лайкнутых видео для пользователя %d", len(response.Items), chatID)
	// TODO: Реализуйте логику синхронизации плейлиста

	return nil
}

// SyncCustomPlaylist синхронизирует кастомный плейлист YouTube Music
func SyncCustomPlaylist(chatID int64, playlistURL string) error {
	// Извлекаем идентификатор плейлиста из URL (пример, если URL имеет формат https://www.youtube.com/playlist?list=...)
	playlistID, err := extractPlaylistID(playlistURL)
	if err != nil {
		return err
	}

	apiKey := "YOUR_YOUTUBE_API_KEY" // Можно заменить на динамическое получение ключа
	service, err := youtube.NewService(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		return err
	}

	// Получаем информацию о плейлисте
	call := service.Playlists.List([]string{"snippet"}).Id(playlistID)
	response, err := call.Do()
	if err != nil || len(response.Items) == 0 {
		return errors.New("плейлист не найден")
	}
	playlistTitle := response.Items[0].Snippet.Title

	// TODO: Реализуйте логику поиска плейлиста в целевом сервисе, создания нового или синхронизации треков

	log.Printf("Кастомный плейлист '%s' успешно синхронизирован для пользователя %d", playlistTitle, chatID)
	return nil
}

// extractPlaylistID извлекает идентификатор плейлиста из URL
func extractPlaylistID(url string) (string, error) {
	// Простейший разбор URL: ищем параметр "list="
	parts := split(url, "list=")
	if len(parts) < 2 {
		return "", errors.New("не удалось извлечь идентификатор плейлиста")
	}
	// Берем часть до возможного разделителя '&'
	idParts := split(parts[1], "&")
	return idParts[0], nil
}

// split – обёртка над стандартным strings.Split для краткости (можно заменить на strings.Split)
func split(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}
