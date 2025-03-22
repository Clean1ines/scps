// pkg/oauth/soundcloud.go
package oauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Clean1ines/scps/pkg/storage"
)

const soundcloudTokenTTL = 24 * time.Hour

type SoundCloudToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"-"`
	TokenType    string    `json:"token_type"`
}

// ExchangeSoundCloudCode обменивает код на токен для SoundCloud.
func ExchangeSoundCloudCode(code string) (*SoundCloudToken, error) {
	redirectURI := os.Getenv("SOUNDCLOUD_REDIRECT_URI")
	clientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")
	clientSecret := os.Getenv("SOUNDCLOUD_CLIENT_SECRET")

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", "https://api.soundcloud.com/oauth2/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SoundCloud exchange error: %s", body)
	}
	var token SoundCloudToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

// RefreshSoundCloudToken обновляет access token для SoundCloud.
func RefreshSoundCloudToken(refreshToken string) (*SoundCloudToken, error) {
	clientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")
	clientSecret := os.Getenv("SOUNDCLOUD_CLIENT_SECRET")
	redirectURI := os.Getenv("SOUNDCLOUD_REDIRECT_URI")

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", "https://api.soundcloud.com/oauth2/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SoundCloud refresh error: %s", body)
	}
	var token SoundCloudToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	token.RefreshToken = refreshToken
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

// StoreSoundCloudToken сохраняет SoundCloud токен в Redis.
func StoreSoundCloudToken(userID int, token *SoundCloudToken) error {
	key := fmt.Sprintf("soundcloud_token:%d", userID)
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return storage.SetValue(key, data, soundcloudTokenTTL)
}

// GetStoredSoundCloudToken получает SoundCloud токен из Redis и обновляет его при необходимости.
func GetStoredSoundCloudToken(userID int) (*SoundCloudToken, error) {
	key := fmt.Sprintf("soundcloud_token:%d", userID)
	data, err := storage.GetValue(key)
	if err != nil {
		return nil, err
	}
	var token SoundCloudToken
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, err
	}
	if time.Now().After(token.ExpiresAt) {
		newToken, err := RefreshSoundCloudToken(token.RefreshToken)
		if err != nil {
			return nil, err
		}
		if err := StoreSoundCloudToken(userID, newToken); err != nil {
			// Здесь можно добавить логирование ошибки
		}
		return newToken, nil
	}
	return &token, nil
}