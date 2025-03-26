package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/Clean1ines/scps/pkg/storage"
)

const (
	rateWindow  = 5 * time.Second
	maxRequests = 3
)

func RateLimit(userID int64) bool {
	ctx := context.Background()
	key := fmt.Sprintf("rate:%d", userID)
	count, _ := storage.Client.Incr(ctx, key).Result()
	if count == 1 {
		storage.Client.Expire(ctx, key, rateWindow)
	}
	return count <= maxRequests
}
