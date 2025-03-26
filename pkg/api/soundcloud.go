// pkg/api/soundcloud.go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Clean1ines/scps/pkg/api/client"
	"github.com/Clean1ines/scps/pkg/oauth"
	"github.com/Clean1ines/scps/pkg/storage"
)

var scClient = client.New(client.DefaultConcurrencyLimit)

// TrackSC представляет данные трека SoundCloud.
type TrackSC struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Duration     int    `json:"duration"` // миллисекунды
	Genre        string `json:"genre"`
	PermalinkURL string `json:"permalink_url"`
}

// ResolveSoundCloudURL разрешает произвольный URL через SoundCloud API /resolve и возвращает числовой ID.
func ResolveSoundCloudURL(oauthToken, originalURL string) (string, error) {
	resolveURL := "https://api.soundcloud.com/resolve"
	params := url.Values{}
	params.Set("url", originalURL)
	params.Set("client_id", oauthToken)

	req, _ := http.NewRequest("GET", fmt.Sprintf("%s?%s", resolveURL, params.Encode()), nil)
	resp, body, err := scClient.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ошибка разрешения SoundCloud URL: %s", body)
	}

	var result struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", result.ID), nil
}

// GetSoundCloudPlaylistTracksAsync получает треки из плейлиста SoundCloud с использованием числового ID и кэшированием.
func GetSoundCloudPlaylistTracksAsync(token *oauth.SoundCloudToken, playlistURL string) ([]TrackSC, error) {
	// Сначала разрешаем URL до числового ID
	resolvedID, err := ResolveSoundCloudURL(token.AccessToken, playlistURL)
	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("soundcloud_playlist:%s", resolvedID)
	if cached, err := storage.GetValue(cacheKey); err == nil && cached != "" {
		var tracks []TrackSC
		if err := json.Unmarshal([]byte(cached), &tracks); err == nil {
			return tracks, nil
		}
	}

	reqURL := fmt.Sprintf("https://api.soundcloud.com/playlists/%s?oauth_token=%s", resolvedID, token.AccessToken)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, _, err := scClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка получения плейлиста SoundCloud: %s", body)
	}
	var result struct {
		Tracks []TrackSC `json:"tracks"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	data, _ := json.Marshal(result.Tracks)
	storage.SetValue(cacheKey, data, 5*time.Minute)
	return result.Tracks, nil
}

// UpdateSoundCloudPlaylist обновляет плейлист SoundCloud путем замены полного списка треков.
// SoundCloud требует передачи полного списка треков для обновления плейлиста.
func UpdateSoundCloudPlaylist(token *oauth.SoundCloudToken, playlistURL string, tracks []TrackSC) error {
	resolvedID, err := ResolveSoundCloudURL(token.AccessToken, playlistURL)
	if err != nil {
		return err
	}

	// Собираем новый список треков (например, список ID)
	var trackIDs []map[string]int
	for _, track := range tracks {
		trackIDs = append(trackIDs, map[string]int{"id": track.ID})
	}
	payload := map[string]interface{}{
		"tracks": trackIDs,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	reqURL := fmt.Sprintf("https://api.soundcloud.com/playlists/%s?oauth_token=%s", resolvedID, token.AccessToken)
	req, err := http.NewRequest("PUT", reqURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, body, err := scClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка обновления плейлиста SoundCloud: %s", body)
	}
	return nil
}

// Client represents an HTTP client with concurrency control.
type Client struct {
	// ...existing fields...
}

// Get sends a GET request to the specified URL.
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, _, err := c.Do(req)
	if httpResp, ok := resp.(*http.Response); ok {
		return httpResp, err
	}
	return nil, fmt.Errorf("invalid response type")
}

func (c *Client) Do(req *http.Request) (any, any, error) {
	panic("unimplemented")
}
