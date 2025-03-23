// pkg/sync/sync.go
package sync

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/Clean1ines/scps/pkg/api"
    "github.com/Clean1ines/scps/pkg/logging"
    "github.com/Clean1ines/scps/pkg/matching"
    "github.com/Clean1ines/scps/pkg/storage"
)

// SyncHandler обрабатывает HTTP-запрос на синхронизацию плейлистов.
// Параметры передаются через query: ?spotify=<playlistID>&youtube=<playlistID>
func SyncHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    spotifyPlaylistID := r.URL.Query().Get("spotify")
    youtubePlaylistID := r.URL.Query().Get("youtube")
    if spotifyPlaylistID == "" || youtubePlaylistID == "" {
        http.Error(w, "Укажите параметры spotify и youtube", 400)
        return
    }
    logger, err := logging.NewLogger(ctx)
    if err != nil {
        http.Error(w, fmt.Sprintf("Ошибка логгера: %v", err), 500)
        return
    }
    redisClient, err := storage.NewRedisClient(os.Getenv("REDIS_ADDRESS"))
    if err != nil {
        http.Error(w, fmt.Sprintf("Ошибка подключения к Redis: %v", err), 500)
        return
    }
    err = RunSync(ctx, redisClient, spotifyPlaylistID, youtubePlaylistID, logger)
    if err != nil {
        http.Error(w, fmt.Sprintf("Ошибка синхронизации: %v", err), 500)
        return
    }
    w.WriteHeader(200)
    w.Write([]byte("Синхронизация завершена успешно"))
}

// RunSync выполняет двустороннюю синхронизацию плейлистов между Spotify и YouTube Music.
func RunSync(ctx context.Context, redisClient *storage.RedisClient, spotifyPlaylistID, youtubePlaylistID string, logger *logging.Logger) error {
    // Получаем плейлист Spotify.
    spotifyTracks, err := api.GetSpotifyPlaylist(ctx, redisClient, spotifyPlaylistID)
    if err != nil {
        return fmt.Errorf("ошибка получения плейлиста Spotify: %v", err)
    }
    // Получаем плейлист YouTube Music.
    youtubeTracks, err := api.GetYouTubePlaylist(ctx, redisClient, youtubePlaylistID, os.Getenv("YOUTUBE_API_KEY"))
    if err != nil {
        return fmt.Errorf("ошибка получения плейлиста YouTube: %v", err)
    }
    // Преобразуем треки для сравнения.
    spotifyMeta := convertToMetadata(spotifyTracks)
    youtubeMeta := convertToMetadata(youtubeTracks)
    // Определяем недостающие треки.
    missingOnYouTube := matching.FindMissingTracks(spotifyMeta, youtubeMeta)
    missingOnSpotify := matching.FindMissingTracks(youtubeMeta, spotifyMeta)
    // Обновляем плейлист на YouTube.
    if len(missingOnYouTube) > 0 {
        if err := api.AddTracksToYouTubePlaylist(ctx, redisClient, youtubePlaylistID, os.Getenv("YOUTUBE_API_KEY"), convertToTracks(missingOnYouTube)); err != nil {
            logger.Errorf("Ошибка добавления треков на YouTube: %v", err)
        }
    }
    // Обновляем плейлист на Spotify.
    if len(missingOnSpotify) > 0 {
        if err := api.AddTracksToSpotifyPlaylist(ctx, redisClient, spotifyPlaylistID, convertToTracks(missingOnSpotify)); err != nil {
            logger.Errorf("Ошибка добавления треков на Spotify: %v", err)
        }
    }
    // Сохраняем отчет о синхронизации в Redis.
    report := map[string]interface{}{
        "timestamp":     time.Now().Unix(),
        "spotify_added": len(missingOnSpotify),
        "youtube_added": len(missingOnYouTube),
    }
    reportJSON, _ := json.Marshal(report)
    redisClient.Set(ctx, "sync_report", reportJSON, 24*time.Hour)
    logger.Infof("Двусторонняя синхронизация завершена успешно")
    return nil
}

// convertToMetadata преобразует список треков в формат TrackMetadata для сравнения.
func convertToMetadata(tracks []api.Track) []matching.TrackMetadata {
    result := []matching.TrackMetadata{}
    for _, t := range tracks {
        result = append(result, matching.TrackMetadata{
            Name:   t.Name,
            Artist: t.Artist,
        })
    }
    return result
}

// convertToTracks преобразует TrackMetadata обратно в формат треков для API.
func convertToTracks(meta []matching.TrackMetadata) []api.Track {
    result := []api.Track{}
    for _, m := range meta {
        result = append(result, api.Track{
            Name:   m.Name,
            Artist: m.Artist,
        })
    }
    return result
}

// RunPeriodicSync запускает синхронизацию каждые 30 минут, используя дефолтные ID из переменных окружения.
func RunPeriodicSync(ctx context.Context, redisClient *storage.RedisClient, logger *logging.Logger) {
    ticker := time.NewTicker(30 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            spotifyID := os.Getenv("DEFAULT_SPOTIFY_PLAYLIST_ID")
            youtubeID := os.Getenv("DEFAULT_YOUTUBE_PLAYLIST_ID")
            if spotifyID != "" && youtubeID != "" {
                if err := RunSync(ctx, redisClient, spotifyID, youtubeID, logger); err != nil {
                    logger.Errorf("Ошибка периодической синхронизации: %v", err)
                }
            }
        }
    }
}