// pkg/oauth/spotify.go
package oauth

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "net/http"
    "time"

    "github.com/Clean1ines/scps/pkg/logging"
    "github.com/go-redis/redis/v8"
)

var (
    spotifyClientID     string
    spotifyClientSecret string
    spotifyRedirectURI  string
    redisClient         *redis.Client
    logger              *logging.Logger
)

const (
    spotifyTokenURL = "https://accounts.spotify.com/api/token"
    stateKeyPrefix  = "spotify_oauth_state:"
)

// generateState генерирует уникальное значение state для конкретного пользователя и сохраняет его в Redis.
func generateState(ctx context.Context, userID string) (string, error) {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    state := base64.URLEncoding.EncodeToString(b)
    key := stateKeyPrefix + userID
    if err := redisClient.Set(ctx, key, state, 10*time.Minute).Err(); err != nil {
        return "", err
    }
    return state, nil
}

// InitSpotify инициализирует параметры OAuth для Spotify.
func InitSpotify(clientID, clientSecret, redirectURI string, r *redis.Client, logg *logging.Logger) {
    spotifyClientID = clientID
    spotifyClientSecret = clientSecret
    spotifyRedirectURI = redirectURI
    redisClient = r
    logger = logg
}

// SpotifyCallbackHandler обрабатывает callback от Spotify OAuth.
func SpotifyCallbackHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    state := r.URL.Query().Get("state")
    // Фиксированный userID "default" используется для демонстрации.
    expectedState, err := redisClient.Get(ctx, stateKeyPrefix+"default").Result()
    if err != nil || state != expectedState {
        http.Error(w, "Неверный state", http.StatusBadRequest)
        logger.Errorf("Spotify: Неверный state: получено %s, ожидалось %s", state, expectedState)
        return
    }
    code := r.URL.Query().Get("code")
    token, err := exchangeSpotifyCode(code)
    if err != nil {
        http.Error(w, "Ошибка обмена кода на токен", http.StatusInternalServerError)
        logger.Errorf("Spotify: Ошибка обмена кода: %v", err)
        return
    }
    tokenJSON, _ := json.Marshal(token)
    redisClient.Set(ctx, "spotify_token", tokenJSON, time.Hour)
    w.Write([]byte("Spotify OAuth успешно завершен"))
}

// exchangeSpotifyCode обменивает код на access_token.
func exchangeSpotifyCode(code string) (map[string]interface{}, error) {
    req, err := http.NewRequest("POST", spotifyTokenURL, nil)
    if err != nil {
        return nil, err
    }
    q := req.URL.Query()
    q.Add("grant_type", "authorization_code")
    q.Add("code", code)
    q.Add("redirect_uri", spotifyRedirectURI)
    req.URL.RawQuery = q.Encode()
    req.SetBasicAuth(spotifyClientID, spotifyClientSecret)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    bodyBytes, _ := ioutil.ReadAll(resp.Body)
    var tokenResp map[string]interface{}
    if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
        return nil, err
    }
    if resp.StatusCode >= 300 {
        return nil, errors.New("Spotify API вернул ошибку")
    }
    return tokenResp, nil
}

// RefreshSpotifyToken обновляет access_token с использованием refresh_token.
func RefreshSpotifyToken(refreshToken string) (map[string]interface{}, error) {
    req, err := http.NewRequest("POST", spotifyTokenURL, nil)
    if err != nil {
        return nil, err
    }
    q := req.URL.Query()
    q.Add("grant_type", "refresh_token")
    q.Add("refresh_token", refreshToken)
    q.Add("redirect_uri", spotifyRedirectURI)
    req.URL.RawQuery = q.Encode()
    req.SetBasicAuth(spotifyClientID, spotifyClientSecret)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    bodyBytes, _ := ioutil.ReadAll(resp.Body)
    var tokenResp map[string]interface{}
    if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
        return nil, err
    }
    if resp.StatusCode >= 300 {
        return nil, errors.New("Spotify API вернул ошибку при обновлении токена")
    }
    return tokenResp, nil
}