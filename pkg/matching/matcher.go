// pkg/matching/matcher.go
package matching

import (
	"strings"

	"github.com/xrash/smetrics"
)

// TrackMetadata содержит расширенные данные трека.
type TrackMetadata struct {
	Title       string
	Artist      string
	Duration    int // длительность в секундах
	Album       string
	ReleaseYear string
}

// FuzzyMatch возвращает процент совпадения между двумя треками, учитывая все поля.
func FuzzyMatch(a, b TrackMetadata) int {
	titleScore := similarity(a.Title, b.Title)
	artistScore := similarity(a.Artist, b.Artist)
	albumScore := similarity(a.Album, b.Album)
	yearScore := similarity(a.ReleaseYear, b.ReleaseYear)
	durationScore := 100
	if abs(a.Duration-b.Duration) > 5 {
		durationScore = 0
	}
	// Весовые коэффициенты: название и исполнитель важнее остальных.
	totalScore := (titleScore*40 + artistScore*30 + albumScore*15 + yearScore*10 + durationScore*5) / 100
	return totalScore
}

func similarity(s1, s2 string) int {
	s1, s2 = strings.ToLower(s1), strings.ToLower(s2)
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}
	if maxLen == 0 {
		return 100
	}
	distance := smetrics.WagnerFischer(s1, s2, 1, 1, 2)
	score := 100 - (distance * 100 / maxLen)
	return score
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
