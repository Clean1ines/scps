// pkg/api/free_recognition.go
package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"
)

type AcoustIDResponse struct {
	Results []struct {
		Score      float64 `json:"score"`
		Recordings []struct {
			Title   string `json:"title"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
			ReleaseGroups []struct {
				Title            string `json:"title"`
				FirstReleaseDate string `json:"first-release-date"`
			} `json:"releasegroups"`
		} `json:"recordings"`
	} `json:"results"`
}

// RecognizeMusicFromFile вычисляет аудиофингерпринт с помощью fpcalc и обращается к AcoustID API.
// Добавлены проверки: если аудиофайл слишком короткий или fpcalc не возвращает корректный результат – генерируется ошибка.
func RecognizeMusicFromFile(filePath string) (*AcoustIDResponse, error) {
	// Проверяем, существует ли файл и имеет ли минимальную длительность (например, 10 секунд)
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("файл не найден: %v", err)
	}
	if info.Size() < 10000 { // условный порог для минимального размера файла
		return nil, fmt.Errorf("файл слишком мал для анализа")
	}

	// Запускаем fpcalc для получения аудиофингерпринта
	cmd := exec.Command("fpcalc", "-json", filePath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("fpcalc error: %v", err)
	}
	var fpData struct {
		Fingerprint string `json:"fingerprint"`
		Duration    int    `json:"duration"`
	}
	if err := json.Unmarshal(out, &fpData); err != nil {
		return nil, fmt.Errorf("fpcalc unmarshal error: %v", err)
	}
	if fpData.Fingerprint == "" || fpData.Duration < 10 {
		return nil, fmt.Errorf("недостаточные данные для распознавания")
	}

	// Обращаемся к AcoustID API для распознавания музыки
	apiKey := os.Getenv("ACOUSTID_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ACOUSTID_API_KEY не задан")
	}
	endpoint := "https://api.acoustid.org/v2/lookup"
	params := url.Values{}
	params.Set("client", apiKey)
	params.Set("meta", "recordings+releasegroups")
	params.Set("duration", fmt.Sprintf("%d", fpData.Duration))
	params.Set("fingerprint", fpData.Fingerprint)
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("AcoustID request error: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("AcoustID read error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AcoustID error: %s", body)
	}
	var acoustidResp AcoustIDResponse
	if err := json.Unmarshal(body, &acoustidResp); err != nil {
		return nil, fmt.Errorf("AcoustID unmarshal error: %v", err)
	}
	return &acoustidResp, nil
}