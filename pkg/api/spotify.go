// pkg/api/spotify.go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Clean1ines/scps/pkg/oauth"
	"github.com/Clean1ines/scps/pkg/storage"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}
var sem = make(chan struct{}, 5) // Ограничение параллельных запросов

// Track представляет расширенные метаданные трека.
type Track struct {
	URI         string
	Title       string
	Artist      string
	Duration    int // секунды
	Album       string
	ReleaseYear string
}

// GetSpotifyPlaylistTracksAsync получает треки плейлиста с пагинацией, параллельно и кэшированием.
// Если данные уже закэшированы в Redis, они возвращаются без повторных запросов.
func GetSpotifyPlaylistTracksAsync(token *oauth.SpotifyToken, playlistID string) ([]Track, error) {
	cacheKey := fmt.Sprintf("spotify_playlist:%s", playlistID)
	if cached, err := storage.GetValue(cacheKey); err == nil && cached != "" {
		var tracks []Track
		if err := json.Unmarshal([]byte(cached), &tracks); err == nil {
			return tracks, nil
		}
	}

	var (
		tracks []Track
		mu     sync.Mutex
		wg     sync.WaitGroup
	)

	// Первый запрос для получения общего количества треков
	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks?limit=1", playlistID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var initResult struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(body, &initResult); err != nil {
		return nil, err
	}
	total := initResult.Total
	limit := 100
	pages := (total + limit - 1) / limit

	// Параллельные запросы с использованием семафора для ограничения количества одновременных запросов
	for p := 0; p < pages; p++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			offset := page * limit
			url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks?limit=%d&offset=%d", playlistID, limit, offset)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Add("Authorization", "Bearer "+token.AccessToken)
			resp, err := httpClient.Do(req)
			if err != nil {
				// Можно добавить логирование через Cloud Logging
				return
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var result struct {
				Items []struct {
					Track struct {
						URI      string `json:"uri"`
						Name     string `json:"name"`
						Duration int    `json:"duration_ms"`
						Album    struct {
							Name        string `json:"name"`
							ReleaseDate string `json:"release_date"`
						} `json:"album"`
						Artists []struct {
							Name string `json:"name"`
						} `json:"artists"`
					} `json:"track"`
				} `json:"items"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return
			}
			var localTracks []Track
			for _, item := range result.Items {
				artistNames := ""
				for i, a := range item.Track.Artists {
					if i > 0 {
						artistNames += ", "
					}
					artistNames += a.Name
				}
				localTracks = append(localTracks, Track{
					URI:         item.Track.URI,
					Title:       item.Track.Name,
					Artist:      artistNames,
					Duration:    item.Track.Duration / 1000,
					Album:       item.Track.Album.Name,
					ReleaseYear: extractYear(item.Track.Album.ReleaseDate),
				})
			}
			mu.Lock()
			tracks = append(tracks, localTracks...)
			mu.Unlock()
		}(p)
		time.Sleep(200 * time.Millisecond) // Контроль частоты запросов для rate-limit
	}
	wg.Wait()

	data, _ := json.Marshal(tracks)
	storage.SetValue(cacheKey, data, 5*time.Minute)
	return tracks, nil
}

func extractYear(dateStr string) string {
	if len(dateStr) >= 4 {
		return dateStr[:4]
	}
	return ""
}

// AddTracksToSpotifyPlaylist добавляет треки в указанный плейлист пакетами.
func AddTracksToSpotifyPlaylist(token *oauth.SpotifyToken, playlistID string, tracks []Track) error {
	batchSize := 100
	for i := 0; i < len(tracks); i += batchSize {
		end := i + batchSize
		if end > len(tracks) {
			end = len(tracks)
		}
		var uris []string
		for _, t := range tracks[i:end] {
			uris = append(uris, t.URI)
		}
		payload := map[string]interface{}{
			"uris": uris,
		}
		payloadBytes, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlistID), bytes.NewReader(payloadBytes))
		req.Header.Add("Authorization", "Bearer "+token.AccessToken)
		req.Header.Add("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			return fmt.Errorf("error adding tracks: %s", body)
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}
