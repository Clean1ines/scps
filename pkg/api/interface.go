package api

import (
	"context"

	"github.com/Clean1ines/scps/pkg/matching"
)

// MusicService определяет интерфейс для работы с музыкальными сервисами
type MusicService interface {
	GetPlaylistTracks(ctx context.Context, playlistURL string) ([]Track, error)
	SearchTrack(ctx context.Context, track Track) (Track, error)
	CreatePlaylist(ctx context.Context, name string) (Playlist, error)
	AddTracksToPlaylist(ctx context.Context, playlistID string, tracks []Track) error
}

// Track определяет общую структуру трека для всех сервисов
type Track struct {
	ID          string
	URI         string // URI трека (например, для Spotify)
	Title       string
	Artist      string
	Album       string
	Duration    int    // длительность в секундах
	ISRC        string // международный стандартный код записи
	ReleaseYear string
}

// ToMetadata конвертирует трек в формат для сравнения
func (t Track) ToMetadata() matching.TrackMetadata {
	return matching.TrackMetadata{
		Title:       t.Title,
		Artist:      t.Artist,
		Album:       t.Album,
		Duration:    t.Duration,
		ReleaseYear: t.ReleaseYear,
	}
}

type Playlist struct {
	ID          string
	Name        string
	Description string
	TrackCount  int
}
