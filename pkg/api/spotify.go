// pkg/api/spotify.go
package api

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"

    "github.com/Clean1ines/scps/pkg/logging"
    "github.com/go-redis/redis/v8"
)

// Track представляет аудиотрек в плейлисте Spotify.
type Track struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Artist string `json:"artist"`
}

// PlaylistResponse представляет ответ Spotify API для плейлиста.
type PlaylistResponse struct {
    Items []Track `json:"items"`
}

// GetSpotifyPlaylist получает треки из плейлиста Spotify по его ID.
func GetSpotifyPlaylist(ctx context.Context, redisClient *redis.Client, playlistID string) ([]Track, error) {
    token, err := redisClient.Get(ctx, "spotify_token").Result()
    if err != nil {
        return nil, fmt.Errorf("не удалось получить токен Spotify: %v", err)
    }
    req, err := http.NewRequest("GET", fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlistID), nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+token)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    var pr PlaylistResponse
    if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
        return nil, err
    }
    return pr.Items, nil
}

// AddTracksToSpotifyPlaylist добавляет треки в плейлист Spotify.
func AddTracksToSpotifyPlaylist(ctx context.Context, redisClient *redis.Client, playlistID string, tracks []Track) error {
    token, err := redisClient.Get(ctx, "spotify_token").Result()
    if err != nil {
        return fmt.Errorf("не удалось получить токен Spotify: %v", err)
    }
    trackURIs := []string{}
    for _, t := range tracks {
        trackURIs = append(trackURIs, "spotify:track:"+t.ID)
    }
    bodyData := map[string]interface{}{
        "uris": trackURIs,
    }
    bodyJSON, _ := json.Marshal(bodyData)
    req, err := http.NewRequest("POST", fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlistID), bytes.NewReader(bodyJSON))
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        return errors.New("ошибка добавления треков на Spotify")
    }
    return nil
}