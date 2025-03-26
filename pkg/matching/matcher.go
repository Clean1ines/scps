// pkg/matching/matcher.go
package matching

import (
	"regexp"
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
	// Нормализация строк перед сравнением
	titleScore := similarity(normalizeString(a.Title), normalizeString(b.Title))
	artistScore := similarity(normalizeString(a.Artist), normalizeString(b.Artist))
	albumScore := similarity(normalizeString(a.Album), normalizeString(b.Album))
	yearScore := similarity(a.ReleaseYear, b.ReleaseYear)

	// Улучшенное сравнение длительности с градацией
	durationScore := calculateDurationScore(a.Duration, b.Duration)

	// Скорректированные веса для более точного сравнения
	return (titleScore*35 + artistScore*35 + durationScore*15 + albumScore*10 + yearScore*5) / 100
}

func similarity(s1, s2 string) int {
	s1, s2 = strings.ToLower(s1), strings.ToLower(s2)

	// Handle empty strings specially
	if s1 == "" && s2 == "" {
		return 100
	}
	if s1 == "" || s2 == "" {
		return 0
	}

	// Get max length for normalization
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	// Calculate Levenshtein distance
	distance := smetrics.WagnerFischer(s1, s2, 1, 1, 2)

	// Convert to similarity score and clamp between 0-100
	score := 100 - ((distance * 100) / maxLen)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

func calculateDurationScore(d1, d2 int) int {
	diff := abs(d1 - d2)
	percentage := (1.0 - float64(diff)/float64(max(d1, d2))) * 100
	switch {
	case percentage >= 95:
		return 100
	case percentage >= 90:
		return 80
	case percentage >= 85:
		return 60
	case percentage >= 80:
		return 40
	case percentage >= 75:
		return 20
	default:
		return 0
	}
}

func normalizeString(s string) string {
	s = strings.ToLower(s)
	// Расширенная нормализация
	replacements := map[string]string{
		"(official)": "",
		"(lyrics)":   "",
		// ...
		"(official video)": "",
		"(lyric video)":    "",
		"(audio)":          "",
		"(original mix)":   "",
		"feat.":            "",
		"ft.":              "",
		"&":                "and",
	}

	for old, new := range replacements {
		s = strings.ReplaceAll(s, old, new)
	}

	// Удаляем все скобки и их содержимое
	s = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(s, "")

	// Удаляем специальные символы
	s = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(s, "")

	return strings.TrimSpace(s)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

type Matcher struct {
	MinimumScore int
}

func NewMatcher() *Matcher {
	return &Matcher{
		MinimumScore: 80,
	}
}

func (m *Matcher) Match(a, b TrackMetadata) int {
	return FuzzyMatch(a, b)
}
