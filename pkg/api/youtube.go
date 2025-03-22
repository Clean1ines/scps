// pkg/api/youtube.go
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/Clean1ines/scps/pkg/oauth"
	"github.com/Clean1ines/scps/pkg/storage"
)

var ytClient = &http.Client{Timeout: 10 * time.Second}
var ytSem = make(chan struct{}, 5) // Ограничение до 5 одновременных запросов

// Video представляет расширенные метаданные видео на YouTube.
type Video struct {
	VideoID     string
	Title       string
	Channel     string
	Duration    int    // длительность в секундах
	CategoryID  string // идентификатор категории (для музыки часто равен "10")
	PublishedAt string
}

// GetYouTubePlaylistVideos получает все видео из плейлиста с полной обработкой пагинации.
func GetYouTubePlaylistVideos(token *oauth.YouTubeToken, playlistID string) ([]Video, error) {
	cacheKey := fmt.Sprintf("youtube_playlist:%s", playlistID)
	if cached, err := storage.GetValue(cacheKey); err == nil && cached != "" {
		var videos []Video
		if err := json.Unmarshal([]byte(cached), &videos); err == nil {
			return videos, nil
		}
	}

	var (
		videos []Video
		mu     sync.Mutex
		wg     sync.WaitGroup
	)
	baseURL := "https://www.googleapis.com/youtube/v3/playlistItems"
	pageToken := ""
	params := url.Values{}
	params.Set("playlistId", playlistID)
	params.Set("part", "snippet,contentDetails")
	params.Set("maxResults", "50")

	// Функция для запроса одной страницы
	fetchPage := func(pt string) error {
		defer wg.Done()
		ytSem <- struct{}{}
		defer func() { <-ytSem }()
		p := url.Values{}
		for k, v := range params {
			p[k] = v
		}
		if pt != "" {
			p.Set("pageToken", pt)
		}
		reqURL := fmt.Sprintf("%s?%s", baseURL, p.Encode())
		req, _ := http.NewRequest("GET", reqURL, nil)
		req.Header.Add("Authorization", "Bearer "+token.AccessToken)
		resp, err := ytClient.Do(req)
		if err != nil {
			return err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("YouTube API error: %s", body)
		}
		var result struct {
			Items         []struct {
				Snippet struct {
					Title        string `json:"title"`
					ChannelTitle string `json:"channelTitle"`
					PublishedAt  string `json:"publishedAt"`
				} `json:"snippet"`
				ContentDetails struct {
					VideoId string `json:"videoId"`
				} `json:"contentDetails"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}
		var localVideos []Video
		for _, item := range result.Items {
			video := Video{
				VideoID:     item.ContentDetails.VideoId,
				Title:       item.Snippet.Title,
				Channel:     item.Snippet.ChannelTitle,
				PublishedAt: item.Snippet.PublishedAt,
			}
			// Получаем дополнительные детали (длительность, категория) для каждого видео
			details, err := getYouTubeVideoDetails(token, video.VideoID)
			if err == nil {
				video.Duration = details.Duration
				video.CategoryID = details.CategoryID
			}
			localVideos = append(localVideos, video)
		}
		mu.Lock()
		videos = append(videos, localVideos...)
		mu.Unlock()
		// Если есть следующая страница, запускаем её обработку
		if result.NextPageToken != "" {
			wg.Add(1)
			go fetchPage(result.NextPageToken)
		}
		return nil
	}

	wg.Add(1)
	go fetchPage(pageToken)
	wg.Wait()

	data, _ := json.Marshal(videos)
	storage.SetValue(cacheKey, data, 5*time.Minute)
	return videos, nil
}

// VideoDetails содержит информацию о длительности и категории видео.
type VideoDetails struct {
	Duration   int    // в секундах
	CategoryID string
}

// getYouTubeVideoDetails запрашивает дополнительные детали для видео через videos.list.
func getYouTubeVideoDetails(token *oauth.YouTubeToken, videoID string) (*VideoDetails, error) {
	detailsURL := "https://www.googleapis.com/youtube/v3/videos"
	params := url.Values{}
	params.Set("id", videoID)
	params.Set("part", "contentDetails")
	reqURL := fmt.Sprintf("%s?%s", detailsURL, params.Encode())
	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	resp, err := ytClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("YouTube details error: %s", body)
	}
	var result struct {
		Items []struct {
			ContentDetails struct {
				Duration   string `json:"duration"`   // ISO 8601 формат, например "PT3M15S"
				CategoryId string `json:"categoryId"` // для музыки часто "10"
			} `json:"contentDetails"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("нет деталей для видео %s", videoID)
	}
	// Преобразуем ISO 8601 длительность в секунды
	durationSec, err := parseISODuration(result.Items[0].ContentDetails.Duration)
	if err != nil {
		return nil, err
	}
	return &VideoDetails{
		Duration:   durationSec,
		CategoryID: result.Items[0].ContentDetails.CategoryId,
	}, nil
}

// parseISODuration парсит ISO 8601 длительность, возвращая длительность в секундах.
func parseISODuration(iso string) (int, error) {
	// Простой парсер, поддерживающий форматы PT#M#S (без часов)
	var minutes, seconds int
	_, err := fmt.Sscanf(iso, "PT%dM%dS", &minutes, &seconds)
	if err != nil {
		// Если формат отличается, можно расширить парсер
		return 0, fmt.Errorf("неподдерживаемый формат длительности: %s", iso)
	}
	return minutes*60 + seconds, nil
}

// AddVideosToYouTubePlaylist добавляет видео в указанный плейлист.
func AddVideosToYouTubePlaylist(token *oauth.YouTubeToken, playlistID string, videoIDs []string) error {
	baseURL := "https://www.googleapis.com/youtube/v3/playlistItems?part=snippet"
	client := &http.Client{Timeout: 10 * time.Second}
	for _, videoID := range videoIDs {
		payload := map[string]interface{}{
			"snippet": map[string]interface{}{
				"playlistId": playlistID,
				"resourceId": map[string]string{
					"kind":    "youtube#video",
					"videoId": videoID,
				},
			},
		}
		payloadBytes, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", baseURL, bytes.NewReader(payloadBytes))
		req.Header.Add("Authorization", "Bearer "+token.AccessToken)
		req.Header.Add("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("ошибка добавления видео: %s", body)
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}