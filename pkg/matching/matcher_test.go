// pkg/matching/matcher_test.go
package matching

import "testing"

func TestFuzzyMatch(t *testing.T) {
	a := TrackMetadata{
		Title:       "Song Title",
		Artist:      "Artist Name",
		Duration:    210,
		Album:       "Album Name",
		ReleaseYear: "2020",
	}
	b := TrackMetadata{
		Title:       "Song Title",
		Artist:      "Artist Name",
		Duration:    213,
		Album:       "Album Name",
		ReleaseYear: "2020",
	}
	score := FuzzyMatch(a, b)
	if score < 90 {
		t.Errorf("Ожидался высокий процент совпадения, получено %d", score)
	}
}