// pkg/api/music_recognition.go
package api

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"

    "github.com/Clean1ines/scps/pkg/logging"
)

// MusicMetadata содержит метаданные трека, полученные от AcoustID.
type MusicMetadata struct {
    Title  string `json:"title"`
    Artist string `json:"artist"`
    Album  string `json:"album"`
}

// RecognizeMusic вычисляет аудиофингерпринт с помощью fpcalc и запрашивает метаданные через AcoustID API.
func RecognizeMusic(audioFilePath string, logger *logging.Logger) (*MusicMetadata, error) {
    // Выполнение fpcalc для получения отпечатка и длительности аудио.
    cmd := exec.Command("fpcalc", "-json", audioFilePath)
    output, err := cmd.Output()
    if err != nil {
        logger.Errorf("Ошибка выполнения fpcalc: %v", err)
        return nil, err
    }
    var fpResult map[string]interface{}
    if err := json.Unmarshal(output, &fpResult); err != nil {
        logger.Errorf("Ошибка парсинга fpcalc: %v", err)
        return nil, err
    }
    duration, ok := fpResult["duration"].(float64)
    if !ok || duration < 30 {
        return nil, fmt.Errorf("аудио слишком короткое или некорректное")
    }
    fingerprint, ok := fpResult["fingerprint"].(string)
    if !ok || strings.TrimSpace(fingerprint) == "" {
        return nil, fmt.Errorf("не удалось получить аудиофингерпринт")
    }
    // Формируем URL для запроса к AcoustID API.
    apiKey := "EMX96S9tia"
    url := fmt.Sprintf("https://api.acoustid.org/v2/lookup?client=%s&duration=%d&fingerprint=%s&meta=recordings", apiKey, int(duration), fingerprint)
    resp, err := exec.Command("curl", "-s", url).Output()
    if err != nil {
        logger.Errorf("Ошибка вызова AcoustID API: %v", err)
        return nil, err
    }
    var acResp struct {
        Results []struct {
            Recordings []struct {
                Title  string `json:"title"`
                Artist string `json:"artist"`
                Album  string `json:"releasegroup"`
            } `json:"recordings"`
        } `json:"results"`
    }
    if err := json.Unmarshal(resp, &acResp); err != nil {
        logger.Errorf("Ошибка парсинга ответа AcoustID: %v", err)
        return nil, err
    }
    if len(acResp.Results) == 0 || len(acResp.Results[0].Recordings) == 0 {
        return nil, fmt.Errorf("нет результатов распознавания")
    }
    rec := acResp.Results[0].Recordings[0]
    metadata := &MusicMetadata{
        Title:  rec.Title,
        Artist: rec.Artist,
        Album:  rec.Album,
    }
    return metadata, nil
}