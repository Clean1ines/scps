// pkg/health/health_test.go
package health

import (
    "io/ioutil"
    "net/http"
    "net/http/httptest"
    "testing"
)

// HealthHandler возвращает ответ "OK" для проверки работоспособности сервера.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

func TestHealthHandler(t *testing.T) {
    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    HealthHandler(w, req)
    resp := w.Result()
    body, _ := ioutil.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Ожидаемый статус 200, получен %d", resp.StatusCode)
    }
    if string(body) != "OK" {
        t.Errorf("Ожидаемое тело 'OK', получено '%s'", string(body))
    }
}