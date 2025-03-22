// pkg/oauth/spotify.go
package oauth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Clean1ines/scps/pkg/storage"
)

const spotifyTokenTTL = 24 * time.Hour

type SpotifyToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"-"`
	TokenType    string    `json:"token_type"`
}

// ExchangeSpotifyCode обменивает код на токен.
func ExchangeSpotifyCode(code string) (*SpotifyToken, error) {
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI")
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", clientID, clientSecret)))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify exchange error: %s", body)
	}
	var token SpotifyToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

// RefreshSpotifyToken обновляет access_token.
func RefreshSpotifyToken(refreshToken string) (*SpotifyToken, error) {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", clientID, clientSecret)))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify refresh error: %s", body)
	}
	var token SpotifyToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	// Обычно refreshToken не меняется
	token.RefreshToken = refreshToken
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

// StoreSpotifyToken сохраняет токен в Redis.
func StoreSpotifyToken(userID int, token *SpotifyToken) error {
	key := fmt.Sprintf("spotify_token:%d", userID)
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return storage.SetValue(key, data, spotifyTokenTTL)
}

// GetStoredSpotifyToken получает токен из Redis и обновляет его при необходимости.
func GetStoredSpotifyToken(userID int) (*SpotifyToken, error) {
	key := fmt.Sprintf("spotify_token:%d", userID)
	data, err := storage.GetValue(key)
	if err != nil {
		return nil, err
	}
	var token SpotifyToken
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, err
	}
	if time.Now().After(token.ExpiresAt) {
		newToken, err := RefreshSpotifyToken(token.RefreshToken)
		if err != nil {
			return nil, err
		}
		if err := StoreSpotifyToken(userID, newToken); err != nil {
			// Логирование обновления токена через Cloud Logging можно добавить здесь
		}
		return newToken, nil
	}
	return &token, nil
}
