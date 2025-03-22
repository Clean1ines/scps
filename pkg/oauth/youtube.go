// pkg/oauth/youtube.go
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

const youtubeTokenTTL = 24 * time.Hour

type YouTubeToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"-"`
	TokenType    string    `json:"token_type"`
}

// ExchangeYouTubeCode обменивает код на токен для YouTube.
func ExchangeYouTubeCode(code string) (*YouTubeToken, error) {
	redirectURI := os.Getenv("YOUTUBE_REDIRECT_URI")
	clientID := os.Getenv("YOUTUBE_CLIENT_ID")
	clientSecret := os.Getenv("YOUTUBE_CLIENT_SECRET")

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/token", bytes.NewBufferString(data.Encode()))
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
		return nil, fmt.Errorf("YouTube exchange error: %s", body)
	}
	var token YouTubeToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

// RefreshYouTubeToken обновляет access token YouTube.
func RefreshYouTubeToken(refreshToken string) (*YouTubeToken, error) {
	clientID := os.Getenv("YOUTUBE_CLIENT_ID")
	clientSecret := os.Getenv("YOUTUBE_CLIENT_SECRET")
	redirectURI := os.Getenv("YOUTUBE_REDIRECT_URI")

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/token", bytes.NewBufferString(data.Encode()))
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
		return nil, fmt.Errorf("YouTube refresh error: %s", body)
	}
	var token YouTubeToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	// Обычно refresh token не меняется
	token.RefreshToken = refreshToken
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

// StoreYouTubeToken сохраняет YouTube токен в Redis.
func StoreYouTubeToken(userID int, token *YouTubeToken) error {
	key := fmt.Sprintf("youtube_token:%d", userID)
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return storage.SetValue(key, data, youtubeTokenTTL)
}

// GetStoredYouTubeToken получает YouTube токен из Redis и обновляет его при необходимости.
func GetStoredYouTubeToken(userID int) (*YouTubeToken, error) {
	key := fmt.Sprintf("youtube_token:%d", userID)
	data, err := storage.GetValue(key)
	if err != nil {
		return nil, err
	}
	var token YouTubeToken
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, err
	}
	if time.Now().After(token.ExpiresAt) {
		newToken, err := RefreshYouTubeToken(token.RefreshToken)
		if err != nil {
			return nil, err
		}
		if err := StoreYouTubeToken(userID, newToken); err != nil {
			// Здесь можно добавить логирование ошибки
		}
		return newToken, nil
	}
	return &token, nil
}
