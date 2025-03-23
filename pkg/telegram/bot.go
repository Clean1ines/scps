// pkg/telegram/bot.go
package telegram

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

    "github.com/Clean1ines/scps/pkg/logging"
    "github.com/Clean1ines/scps/pkg/oauth"
    "github.com/Clean1ines/scps/pkg/pubsub"
    "github.com/Clean1ines/scps/pkg/sync"
    "github.com/go-redis/redis/v8"
)

// Константы состояний диалога
const (
    StateIdle           = "idle"
    StateAwaitSource    = "await_source"  // Ожидание выбора исходного сервиса
    StateAwaitURL       = "await_url"     // Ожидание ввода URL плейлиста
    StateAwaitTarget    = "await_target"  // Ожидание выбора целевого сервиса
    StateSyncInProgress = "sync_in_progress"
    StateSyncCompleted  = "sync_completed"
)

// SessionKey формирует ключ для хранения сессии пользователя в Redis.
func SessionKey(chatID int64) string {
    return fmt.Sprintf("session:%d", chatID)
}

// Session хранит данные текущей сессии пользователя.
type Session struct {
    State          string `json:"state"`
    SourcePlatform string `json:"source_platform"` // "spotify" или "youtube"
    PlaylistURL    string `json:"playlist_url"`    // URL исходного плейлиста
    TargetPlatform string `json:"target_platform"` // "spotify" или "youtube"
}

// Bot представляет Telegram-бота.
type Bot struct {
    api         *tgbotapi.BotAPI
    redisClient *redis.Client
    logger      *logging.Logger
    psClient    *pubsub.PubSubClient
}

// NewBot создает нового Telegram-бота.
func NewBot(token string, r *redis.Client, logger *logging.Logger, psClient *pubsub.PubSubClient) (*Bot, error) {
    api, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        return nil, err
    }
    return &Bot{
        api:         api,
        redisClient: r,
        logger:      logger,
        psClient:    psClient,
    }, nil
}

// Start запускает получение обновлений от Telegram.
func (b *Bot) Start() {
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60
    updates := b.api.GetUpdatesChan(u)
    for update := range updates {
        if update.Message != nil {
            go b.handleMessage(update.Message)
        }
        if update.CallbackQuery != nil {
            go b.handleCallback(update.CallbackQuery)
        }
    }
}

// handleMessage обрабатывает входящие сообщения и управляет диалогом.
func (b *Bot) handleMessage(msg *tgbotapi.Message) {
    chatID := msg.Chat.ID
    ctx := msg.Context()
    session, _ := b.getSession(ctx, chatID)
    if msg.IsCommand() {
        switch msg.Command() {
        case "start":
            session = &Session{State: StateAwaitSource}
            b.saveSession(ctx, chatID, session)
            b.sendSourceSelection(chatID)
        case "restart":
            session = &Session{State: StateAwaitSource}
            b.saveSession(ctx, chatID, session)
            b.sendSourceSelection(chatID)
        case "report":
            b.sendSyncReport(chatID)
        case "refresh":
            b.refreshToken(ctx, chatID)
        default:
            b.sendText(chatID, "Неизвестная команда. Используйте /start для начала.")
        }
        return
    }
    // Если сообщение не является командой, обрабатываем его согласно состоянию.
    switch session.State {
    case StateAwaitURL:
        session.PlaylistURL = msg.Text
        session.State = StateAwaitTarget
        b.saveSession(ctx, chatID, session)
        b.sendTargetSelection(chatID)
    default:
        b.sendText(chatID, "Пожалуйста, используйте /start для начала работы бота.")
    }
}

// handleCallback обрабатывает нажатия на inline-кнопки.
func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
    chatID := cb.Message.Chat.ID
    ctx := context.Background()
    session, _ := b.getSession(ctx, chatID)
    data := cb.Data
    switch session.State {
    case StateAwaitSource:
        if data == "source_spotify" || data == "source_youtube" {
            session.SourcePlatform = strings.TrimPrefix(data, "source_")
            session.State = StateAwaitURL
            b.saveSession(ctx, chatID, session)
            b.sendText(chatID, "Введите URL плейлиста для синхронизации")
        }
    case StateAwaitTarget:
        if data == "target_spotify" || data == "target_youtube" {
            session.TargetPlatform = strings.TrimPrefix(data, "target_")
            session.State = StateSyncInProgress
            b.saveSession(ctx, chatID, session)
            b.sendText(chatID, "Запуск синхронизации...")
            go b.runSync(ctx, chatID, session)
        }
    case StateSyncCompleted:
        if data == "restart" {
            session = &Session{State: StateAwaitSource}
            b.saveSession(ctx, chatID, session)
            b.sendSourceSelection(chatID)
        }
    default:
        b.sendText(chatID, "Состояние не определено. Используйте /start для начала.")
    }
    callback := tgbotapi.NewCallback(cb.ID, "")
    b.api.Request(callback)
}

// runSync инициирует двустороннюю синхронизацию плейлистов.
func (b *Bot) runSync(ctx context.Context, chatID int64, session *Session) {
    // Извлекаем идентификаторы плейлистов из URL.
    // Здесь должна быть реализация парсинга URL в соответствии с форматами Spotify и YouTube.
    // В этой реализации мы просто берем последний сегмент URL.
    spotifyID := extractPlaylistID(session.PlaylistURL, session.SourcePlatform, "spotify")
    youtubeID := extractPlaylistID(session.PlaylistURL, session.SourcePlatform, "youtube")
    // Если исходный сервис совпадает с целевым, обновляется существующий плейлист.
    if session.TargetPlatform == "spotify" {
        // Если пользователь выбрал Spotify как целевую платформу, используем spotifyID.
    } else if session.TargetPlatform == "youtube" {
        // Если выбран YouTube как целевая, используем youtubeID.
    }
    // Формируем задачу синхронизации.
    task := pubsub.SyncTask{
        Type:              "sync_playlist",
        SpotifyPlaylistID: spotifyID,
        YouTubePlaylistID: youtubeID,
        ChatID:            chatID,
    }
    if err := b.psClient.PublishTask(ctx, task); err != nil {
        b.logger.Errorf("Ошибка публикации задачи: %v", err)
        b.sendText(chatID, fmt.Sprintf("Ошибка запуска синхронизации: %v", err))
        return
    }
    b.sendText(chatID, "Синхронизация запущена. По завершении вы получите отчет.")
    session.State = StateSyncCompleted
    b.saveSession(ctx, chatID, session)
    b.sendRestartButton(chatID)
}

// refreshToken обновляет Spotify access_token по команде /refresh.
func (b *Bot) refreshToken(ctx context.Context, chatID int64) {
    tokenJSON, err := b.redisClient.Get(ctx, "spotify_token").Result()
    if err != nil {
        b.sendText(chatID, "Ошибка получения токена")
        return
    }
    var tokenData map[string]interface{}
    json.Unmarshal([]byte(tokenJSON), &tokenData)
    refreshToken, ok := tokenData["refresh_token"].(string)
    if !ok || refreshToken == "" {
        b.sendText(chatID, "Нет refresh_token")
        return
    }
    newToken, err := oauth.RefreshSpotifyToken(refreshToken)
    if err != nil {
        b.sendText(chatID, "Ошибка обновления токена")
        return
    }
    newJSON, _ := json.Marshal(newToken)
    b.redisClient.Set(ctx, "spotify_token", newJSON, time.Hour)
    b.sendText(chatID, "Токен обновлен")
}

// extractPlaylistID извлекает идентификатор плейлиста из URL для данной платформы.
func extractPlaylistID(url, sourcePlatform, targetPlatform string) string {
    // Реализация должна учитывать форматы URL Spotify и YouTube.
    // Здесь просто берется последний сегмент строки.
    parts := strings.Split(url, "/")
    if len(parts) > 0 {
        return parts[len(parts)-1]
    }
    return ""
}

// saveSession сохраняет состояние сессии в Redis.
func (b *Bot) saveSession(ctx context.Context, chatID int64, session *Session) {
    data, _ := json.Marshal(session)
    b.redisClient.Set(ctx, SessionKey(chatID), data, 24*time.Hour)
}

// getSession получает состояние сессии из Redis.
func (b *Bot) getSession(ctx context.Context, chatID int64) (*Session, error) {
    data, err := b.redisClient.Get(ctx, SessionKey(chatID)).Result()
    if err != nil {
        return &Session{State: StateIdle}, nil
    }
    var session Session
    if err := json.Unmarshal([]byte(data), &session); err != nil {
        return &Session{State: StateIdle}, err
    }
    return &session, nil
}

// sendSourceSelection отправляет кнопки для выбора исходного сервиса.
func (b *Bot) sendSourceSelection(chatID int64) {
    msg := tgbotapi.NewMessage(chatID, "Выберите источник плейлиста:")
    buttons := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("Spotify", "source_spotify"),
            tgbotapi.NewInlineKeyboardButtonData("YouTube", "source_youtube"),
        ),
    )
    msg.ReplyMarkup = buttons
    b.api.Send(msg)
}

// sendTargetSelection отправляет кнопки для выбора целевого сервиса.
func (b *Bot) sendTargetSelection(chatID int64) {
    msg := tgbotapi.NewMessage(chatID, "Выберите целевой сервис для синхронизации:")
    buttons := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("Spotify", "target_spotify"),
            tgbotapi.NewInlineKeyboardButtonData("YouTube", "target_youtube"),
        ),
    )
    msg.ReplyMarkup = buttons
    b.api.Send(msg)
}

// sendRestartButton отправляет кнопку Restart для перезапуска диалога.
func (b *Bot) sendRestartButton(chatID int64) {
    msg := tgbotapi.NewMessage(chatID, "Нажмите Restart для перезапуска бота")
    buttons := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("Restart", "restart"),
        ),
    )
    msg.ReplyMarkup = buttons
    b.api.Send(msg)
}

// sendText отправляет текстовое сообщение пользователю.
func (b *Bot) sendText(chatID int64, text string) {
    msg := tgbotapi.NewMessage(chatID, text)
    b.api.Send(msg)
}

// sendSyncReport получает отчет синхронизации из Redis и отправляет его пользователю.
func (b *Bot) sendSyncReport(chatID int64) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    key := fmt.Sprintf("sync_report_%d", chatID)
    data, err := b.redisClient.Get(ctx, key).Result()
    if err != nil {
        b.logger.Errorf("Не удалось получить отчет: %v", err)
        b.sendText(chatID, "Отчет не найден")
        return
    }
    var report SyncReport
    if err := json.Unmarshal([]byte(data), &report); err != nil {
        b.logger.Errorf("Ошибка разбора отчета: %v", err)
        b.sendText(chatID, "Ошибка формирования отчета")
        return
    }
    msgText := fmt.Sprintf("Синхронизация завершена.\nДобавлено на Spotify: %d треков\nДобавлено на YouTube: %d треков\n", report.SuccessCount, report.SuccessCount)
    if len(report.Errors) == 0 {
        msgText += "Ошибок не обнаружено."
    } else {
        msgText += "Ошибки:\n"
        for _, errMsg := range report.Errors {
            msgText += "- " + errMsg + "\n"
        }
    }
    b.sendText(chatID, msgText)
}