package oauth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Clean1ines/scps/pkg/storage"
)

const (
	defaultTokenTTL = 24 * time.Hour
)

type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"-"`
	TokenType    string    `json:"token_type"`
}

func StoreToken(service string, userID int, token *Token) error {
	key := fmt.Sprintf("%s_token:%d", service, userID)
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return storage.SetValue(key, data, defaultTokenTTL)
}

func GetStoredToken(service string, userID int, forceRefresh bool) (*Token, error) {
	key := fmt.Sprintf("%s_token:%d", service, userID)
	data, err := storage.GetValue(key)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	var token Token
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	// Проверяем необходимость обновления токена
	if forceRefresh || time.Until(token.ExpiresAt) < 10*time.Minute {
		return refreshTokenWithRetry(service, userID, &token)
	}

	return &token, nil
}

func refreshTokenWithRetry(service string, userID int, token *Token) (*Token, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		newToken, err := refreshToken(service, token.RefreshToken)
		if err == nil {
			// Сохраняем новый токен атомарно
			if err := storage.Transaction(func() error {
				return StoreToken(service, userID, newToken)
			}); err == nil {
				return newToken, nil
			}
		}
		lastErr = err
		time.Sleep(time.Second * time.Duration(attempt+1))
	}
	return nil, fmt.Errorf("failed to refresh token after retries: %w", lastErr)
}

func refreshToken(service, refreshToken string) (*Token, error) {
	switch service {
	case "spotify":
		spotifyToken, err := RefreshSpotifyToken(refreshToken)
		if err != nil {
			return nil, err
		}
		return &Token{
			AccessToken:  spotifyToken.AccessToken,
			RefreshToken: spotifyToken.RefreshToken,
			ExpiresIn:    spotifyToken.ExpiresIn,
			ExpiresAt:    spotifyToken.ExpiresAt,
			TokenType:    spotifyToken.TokenType,
		}, nil
	case "youtube":
		ytToken, err := RefreshYouTubeToken(refreshToken)
		if err != nil {
			return nil, err
		}
		return &Token{
			AccessToken:  ytToken.AccessToken,
			RefreshToken: ytToken.RefreshToken,
			ExpiresIn:    ytToken.ExpiresIn,
			ExpiresAt:    ytToken.ExpiresAt,
			TokenType:    ytToken.TokenType,
		}, nil
	case "soundcloud":
		scToken, err := RefreshSoundCloudToken(refreshToken)
		if err != nil {
			return nil, err
		}
		return &Token{
			AccessToken:  scToken.AccessToken,
			RefreshToken: scToken.RefreshToken,
			ExpiresIn:    scToken.ExpiresIn,
			ExpiresAt:    scToken.ExpiresAt,
			TokenType:    scToken.TokenType,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported service: %s", service)
	}
}

func GetAnyStoredToken(userID int64) (*Token, error) {
	services := []string{"spotify", "youtube", "soundcloud"}
	for _, service := range services {
		if token, err := GetStoredToken(service, int(userID), false); err == nil {
			return token, nil
		}
	}
	return nil, fmt.Errorf("no valid tokens found")
}
