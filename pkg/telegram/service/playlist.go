package service

import (
	"context"
	"fmt"

	"github.com/Clean1ines/scps/pkg/api"
	"github.com/Clean1ines/scps/pkg/matching"
	"github.com/Clean1ines/scps/pkg/pubsub"
)

type PlaylistService struct {
	pubsubClient *pubsub.PubSubClient // Changed from Client to PubSubClient
	api          map[string]api.MusicService
	matcher      *matching.Matcher
}

func NewPlaylistService(pubsub *pubsub.PubSubClient) *PlaylistService {
	return &PlaylistService{
		pubsubClient: pubsub,
		api: map[string]api.MusicService{
			"spotify":    api.NewSpotifyService(),
			"youtube":    api.NewYouTubeService(),
			"soundcloud": api.NewSoundCloudService(),
		},
		matcher: matching.NewMatcher(),
	}
}

func (s *PlaylistService) SyncPlaylists(ctx context.Context, task pubsub.Task) error {
	// Получаем треки из исходного плейлиста
	sourceTracks, err := s.api[task.SourceService].GetPlaylistTracks(ctx, task.PlaylistURL)
	if err != nil {
		return fmt.Errorf("failed to get source tracks: %w", err)
	}

	// Создаем новый плейлист в целевом сервисе
	targetPlaylist, err := s.api[task.TargetService].CreatePlaylist(ctx, fmt.Sprintf("Imported from %s", task.SourceService))
	if err != nil {
		return fmt.Errorf("failed to create target playlist: %w", err)
	}

	// Конвертируем треки
	var targetTracks []api.Track
	for _, sourceTrack := range sourceTracks {
		// Ищем соответствующий трек в целевом сервисе
		targetTrack, err := s.api[task.TargetService].SearchTrack(ctx, sourceTrack)
		if err != nil {
			continue // Пропускаем трек, если не нашли
		}

		// Проверяем соответствие с помощью матчера
		score := s.matcher.Match(sourceTrack.ToMetadata(), targetTrack.ToMetadata()) // Fix Metadata calls
		if score >= 80 {                                                             // Минимальный порог соответствия
			targetTracks = append(targetTracks, targetTrack)
		}
	}

	// Добавляем треки в целевой плейлист
	if err := s.api[task.TargetService].AddTracksToPlaylist(ctx, targetPlaylist.ID, targetTracks); err != nil {
		return fmt.Errorf("failed to add tracks: %w", err)
	}

	// Отправляем уведомление пользователю
	s.notifyUser(task.UserID, fmt.Sprintf("Playlist synced! Found %d/%d tracks", len(targetTracks), len(sourceTracks)))
	return nil
}

func (s *PlaylistService) notifyUser(userID int64, message string) {
	// Отправка уведомления через Telegram
	// Реализация зависит от конкретной структуры вашего бота
}

func ProcessPlaylistSync(task pubsub.Task) error {
	switch task.SourceService {
	case "spotify":
		return handleSpotifySync(task)
	case "youtube":
		return handleYouTubeSync(task)
	case "soundcloud":
		return handleSoundCloudSync(task)
	default:
		return fmt.Errorf("unsupported source service: %s", task.SourceService)
	}
}

func handleSpotifySync(task pubsub.Task) error {
	// TODO: Implement Spotify sync
	_ = task // Silence unused parameter until implementation
	return nil
}

func handleYouTubeSync(task pubsub.Task) error {
	// TODO: Implement YouTube sync
	_ = task // Silence unused parameter until implementation
	return nil
}

func handleSoundCloudSync(task pubsub.Task) error {
	// TODO: Implement SoundCloud sync
	_ = task // Silence unused parameter until implementation
	return nil
}
