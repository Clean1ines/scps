// pkg/matching/matching.go
package matching

import (
    "math"
    "strings"

    "github.com/agnivade/levenshtein"
)

// TrackMetadata представляет метаданные трека для сравнения.
type TrackMetadata struct {
    Name   string
    Artist string
}

// FindMissingTracks сравнивает два плейлиста и возвращает треки, отсутствующие в целевом.
func FindMissingTracks(sourcePlaylist, targetPlaylist []TrackMetadata) []TrackMetadata {
    missing := []TrackMetadata{}
    for _, src := range sourcePlaylist {
        found := false
        for _, tgt := range targetPlaylist {
            if areTracksMatching(src, tgt) {
                found = true
                break
            }
        }
        if !found {
            missing = append(missing, src)
        }
    }
    return missing
}

// areTracksMatching сравнивает два трека с использованием нормализации и расстояния Левенштейна.
func areTracksMatching(a, b TrackMetadata) bool {
    aName := strings.ToLower(strings.TrimSpace(a.Name))
    bName := strings.ToLower(strings.TrimSpace(b.Name))
    aArtist := strings.ToLower(strings.TrimSpace(a.Artist))
    bArtist := strings.ToLower(strings.TrimSpace(b.Artist))
    nameDiff := levenshtein.ComputeDistance(aName, bName)
    artistDiff := levenshtein.ComputeDistance(aArtist, bArtist)
    // Если разница в названии и исполнителе меньше 3 символов, считаем треки совпадающими.
    return nameDiff < 3 && artistDiff < 3
}