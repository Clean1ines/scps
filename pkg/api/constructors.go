package api

import (
	"context"
)

func NewSpotifyService() MusicService    { return &SpotifyService{} }
func NewYouTubeService() MusicService    { return &YouTubeService{} }
func NewSoundCloudService() MusicService { return &SoundCloudService{} }

type baseService struct{}

func (s *baseService) GetPlaylistTracks(ctx context.Context, playlistURL string) ([]Track, error) {
	return nil, nil
}

func (s *baseService) SearchTrack(ctx context.Context, track Track) (Track, error) {
	return Track{}, nil
}

func (s *baseService) CreatePlaylist(ctx context.Context, name string) (Playlist, error) {
	return Playlist{}, nil
}

func (s *baseService) AddTracksToPlaylist(ctx context.Context, playlistID string, tracks []Track) error {
	return nil
}

type SpotifyService struct{ baseService }
type YouTubeService struct{ baseService }
type SoundCloudService struct{ baseService }
