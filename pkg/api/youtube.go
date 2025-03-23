// pkg/api/youtube.go
package api

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "errors"

    "github.com/Clean1ines/scps/pkg/logging"
    "github.com/go-redis/redis/v8"
)

// Track представляет видео-трек в плейлисте YouTube.
type Track struct {
    VideoID string `json:"videoId"`
    Title   string `json:"title"`
    Channel string `json:"channelTitle"`
}

// PlaylistResponse представляет ответ YouTube API для плейлиста.
type PlaylistResponse struct {
    Items []struct {
        Snippet struct {
            ResourceId struct {
                VideoId string `json:"videoId"`
            } `json:"resourceId"`
            Title        string `json:"title"`
            ChannelTitle string `json:"channelTitle"`
        } `json:"snippet"`
    } `json:"items"`
}

// GetYouTubePlaylist получает треки из плейлиста YouTube по ID.
func GetYouTubePlaylist(ctx context.Context, redisClient *redis.Client, playlistID, apiKey string) ([]Track, error) {
    token, err := redisClient.Get(ctx, "youtube_token").Result()
    if err != nil {
        return nil, fmt.Errorf("не удалось получить токен YouTube: %v", err)
    }
    url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&playlistId=%s&key=%s", playlistID, apiKey)
    req, err := http.NewRequest("GET", url, nil)
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
    tracks := []Track{}
    for _, item := range pr.Items {
        tracks = append(tracks, Track{
            VideoID: item.Snippet.ResourceId.VideoId,
            Title:   item.Snippet.Title,
            Channel: item.Snippet.ChannelTitle,
        })
    }
    return tracks, nil
}

// AddTracksToYouTubePlaylist добавляет треки в плейлист YouTube.
func AddTracksToYouTubePlaylist(ctx context.Context, redisClient *redis.Client, playlistID, apiKey string, tracks []Track) error {
    token, err := redisClient.Get(ctx, "youtube_token").Result()
    if err != nil {
        return fmt.Errorf("не удалось получить токен YouTube: %v", err)
    }
    for _, track := range tracks {
        bodyData := map[string]interface{}{
            "snippet": map[string]interface{}{
                "playlistId": playlistID,
                "resourceId": map[string]string{
                    "kind":    "youtube#video",
                    "videoId": track.VideoID,
                },
            },
        }
        bodyJSON, _ := json.Marshal(bodyData)
        req, err := http.NewRequest("POST", "https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&key="+apiKey, bytes.NewReader(bodyJSON))
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
            return errors.New(fmt.Sprintf("ошибка добавления видео на YouTube, статус: %d", resp.StatusCode))
        }
    }
    return nil
}