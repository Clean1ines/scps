package middleware

import (
	"github.com/Clean1ines/scps/pkg/oauth"
)

func CheckAuth(userID int64) bool {
	// Check if user has any valid token
	_, err := oauth.GetAnyStoredToken(userID)
	return err == nil
}
