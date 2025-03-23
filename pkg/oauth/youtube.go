// pkg/oauth/youtube.go
package oauth

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "time"

    "github.com/Clean1ines/scps/pkg/logging"
    "github.com/go-redis/redis/v8"
)

var (
    youtubeClientID     string
    youtubeClientSecret string
    youtubeRedirectURI  string
    redisYTClient       *redis.Client
    loggerYT            *logging.Logger
)

const (
    youtubeTokenURL = "https://oauth2.googleapis.com/token"
    stateKeyYT      = "youtube_oauth_state:default"
)

// InitYouTube инициализирует параметры OAuth для YouTube.
func InitYouTube(clientID, clientSecret, redirectURI string, r *redis.Client, logg *logging.Logger) {
    youtubeClientID = clientID
    youtubeClientSecret = clientSecret
    youtubeRedirectURI = redirectURI
    redisYTClient = r
    loggerYT = logg
}

// YouTubeCallbackHandler обрабатывает callback от YouTube OAuth.
func YouTubeCallbackHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    state := r.URL.Query().Get("state")
    expectedState, err := redisYTClient.Get(ctx, stateKeyYT).Result()
    if err != nil || state != expectedState {
        http.Error(w, "Неверный state", http.StatusBadRequest)
        loggerYT.Errorf("YouTube: Неверный state: получено %s, ожидалось %s", state, expectedState)
        return
    }
    code := r.URL.Query().Get("code")
    token, err := exchangeYouTubeCode(code)
    if err != nil {
        http.Error(w, "Ошибка обмена кода на токен", http.StatusInternalServerError)
        loggerYT.Errorf("YouTube: Ошибка обмена кода: %v", err)
        return
    }
    tokenJSON, _ := json.Marshal(token)
    redisYTClient.Set(ctx, "youtube_token", tokenJSON, time.Hour)
    w.Write([]byte("YouTube OAuth успешно завершен"))
}

// exchangeYouTubeCode обменивает код на access_token для YouTube.
func exchangeYouTubeCode(code string) (map[string]interface{}, error) {
    req, err := http.NewRequest("POST", youtubeTokenURL, nil)
    if err != nil {
        return nil, err
    }
    q := req.URL.Query()
    q.Add("grant_type", "authorization_code")
    q.Add("code", code)
    q.Add("redirect_uri", youtubeRedirectURI)
    q.Add("client_id", youtubeClientID)
    q.Add("client_secret", youtubeClientSecret)
    req.URL.RawQuery = q.Encode()
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
        return nil, fmt.Errorf("YouTube API вернул ошибку")
    }
    return tokenResp, nil
}