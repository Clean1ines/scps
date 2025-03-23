// pkg/oauth/spotify_test.go
package oauth

import (
    "testing"
)

func TestExchangeSpotifyCode(t *testing.T) {
    _, err := exchangeSpotifyCode("dummy_code")
    if err == nil {
        t.Errorf("Ожидается ошибка при передаче dummy_code")
    }
}

func TestGenerateState(t *testing.T) {
    state, err := generateState(nil, "default")
    if err != nil {
        t.Errorf("Ошибка генерации state: %v", err)
    }
    if len(state) == 0 {
        t.Errorf("Ожидается непустой state")
    }
}