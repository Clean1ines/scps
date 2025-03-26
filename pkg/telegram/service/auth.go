package service

import (
	"fmt"
	"os"
	"strings"

	"github.com/Clean1ines/scps/pkg/oauth"
)

type Storage interface {
	StoreToken(service string, userID int64, token interface{}) error
}

type AuthService struct {
	storage  Storage
	authURLs map[string]string
}

func NewAuthService() *AuthService {
	return &AuthService{
		authURLs: map[string]string{
			"spotify":    "https://accounts.spotify.com/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=playlist-read-private playlist-modify-public playlist-modify-private user-library-read&state=%d",
			"youtube":    "https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&response_type=code&redirect_uri=%s&scope=https://www.googleapis.com/auth/youtube&state=%d",
			"soundcloud": "https://soundcloud.com/connect?client_id=%s&response_type=code&redirect_uri=%s&scope=non-expiring&state=%d",
		},
	}
}

func (s *AuthService) GetAuthorizationURL(service string, userID int64) (string, error) {
	clientID := os.Getenv(fmt.Sprintf("%s_CLIENT_ID", strings.ToUpper(service)))
	redirectURI := os.Getenv(fmt.Sprintf("%s_REDIRECT_URI", strings.ToUpper(service)))

	if urlTemplate, ok := s.authURLs[service]; ok {
		return fmt.Sprintf(urlTemplate, clientID, redirectURI, userID), nil
	}
	return "", fmt.Errorf("unsupported service: %s", service)
}

func (s *AuthService) HandleAuthCallback(service string, code string, userID int64) error {
	var token interface{}
	var err error

	switch service {
	case "spotify":
		token, err = oauth.ExchangeSpotifyCode(code)
	case "youtube":
		token, err = oauth.ExchangeYouTubeCode(code)
	case "soundcloud":
		token, err = oauth.ExchangeSoundCloudCode(code)
	default:
		return fmt.Errorf("unsupported service: %s", service)
	}

	if err != nil {
		return fmt.Errorf("token exchange failed: %w", err)
	}

	return s.storage.StoreToken(service, userID, token)
}
