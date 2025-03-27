package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Clean1ines/scps/pkg/api"
	"github.com/Clean1ines/scps/pkg/logging"
	"github.com/Clean1ines/scps/pkg/oauth"
	"github.com/Clean1ines/scps/pkg/pubsub"
)

type Logger interface {
	Printf(format string, v ...interface{})
}

type PlaylistProcessor struct {
	logger       logging.LogEntry
	pubsubClient pubsub.Client
}

func NewPlaylistProcessor(pubsubClient pubsub.Client) *PlaylistProcessor {
	return &PlaylistProcessor{
		logger:       *logging.DefaultLogger.StandardLogger(logging.Info),
		pubsubClient: pubsubClient,
	}
}

func (p *PlaylistProcessor) ProcessPlaylist(userID int64, playlistURL string, source, target string) error {
	if !p.validateURL(playlistURL, source) {
		return fmt.Errorf("URL не соответствует выбранному сервису")
	}

	if !p.hasValidTokens(userID, source, target) {
		return fmt.Errorf("отсутствуют необходимые токены авторизации")
	}

	if p.isLikedPlaylist(playlistURL) {
		return p.handleLikedPlaylist(userID, source, target, playlistURL)
	}

	return p.handleCustomPlaylist(userID, source, target, playlistURL)
}

func (p *PlaylistProcessor) handleLikedPlaylist(userID int64, source, target, playlistURL string) error {
	var err error
	switch {
	case source == "spotify" && target == "spotify":
		err = api.SyncLikedSpotify(userID, playlistURL)
	case source == "youtube" && target == "youtube":
		err = api.SyncLikedYouTube(userID, playlistURL)
	case source == "soundcloud" && target == "soundcloud":
		err = api.SyncLikedSoundCloud(userID, playlistURL)
	default:
		err = fmt.Errorf("неподдерживаемая комбинация сервисов для liked playlist")
	}

	if err != nil {
		p.logger.Printf("Ошибка синхронизации liked для userID=%d: %v", userID, err)
	}
	return err
}

func (p *PlaylistProcessor) handleCustomPlaylist(userID int64, source, target, playlistURL string) error {
	ctx := context.Background()
	task := pubsub.Task{
		UserID:        userID,
		SourceService: source,
		TargetService: target,
		PlaylistURL:   playlistURL,
		Action:        "sync-custom",
	}

	return p.pubsubClient.PublishTask(ctx, task)
}

// Вспомогательные методы
func (p *PlaylistProcessor) validateURL(url, source string) bool {
	service, err := detectServiceFromURL(url)
	return err == nil && service == source
}

func (p *PlaylistProcessor) hasValidTokens(userID int64, source, target string) bool {
	_, err1 := oauth.GetStoredToken(source, int(userID), false)
	_, err2 := oauth.GetStoredToken(target, int(userID), false)
	return err1 == nil && err2 == nil
}

func (p *PlaylistProcessor) isLikedPlaylist(url string) bool {
	return strings.Contains(strings.ToLower(url), "liked")
}

func detectServiceFromURL(url string) (string, error) {
	switch {
	case strings.Contains(url, "spotify.com"):
		return "spotify", nil
	case strings.Contains(url, "youtube.com"):
		return "youtube", nil
	case strings.Contains(url, "soundcloud.com"):
		return "soundcloud", nil
	default:
		return "", fmt.Errorf("неизвестный сервис")
	}
}
