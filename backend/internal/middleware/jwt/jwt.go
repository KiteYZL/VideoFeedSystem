package jwt

import (
	"demo/internal/account"
	"demo/internal/auth"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func JWTAuth(accountRepo *account.AccountRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		if accountRepo == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "account repository not initialized"})
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少认证信息"})
			return
		}

		tokenString, err := extractBearerToken(authHeader)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "认证格式错误"})
			return
		}

		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
			return
		}

		accountInfo, err := accountRepo.FindByID(c.Request.Context(), claims.AccountID)
		if err != nil || accountInfo == nil || accountInfo.Token == "" || accountInfo.Token != tokenString {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
			return
		}

		c.Set("account_id", accountInfo.ID)
		c.Set("username", accountInfo.Username)
		c.Next()
	}
}

func SoftJWTAuth(accountRepo *account.AccountRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if strings.TrimSpace(authHeader) == "" {
			c.Next()
			return
		}

		if accountRepo == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "account repository not initialized"})
			return
		}

		tokenString, err := extractBearerToken(authHeader)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "认证格式错误"})
			return
		}

		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
			return
		}

		accountInfo, err := accountRepo.FindByID(c.Request.Context(), claims.AccountID)
		if err != nil || accountInfo == nil || accountInfo.Token == "" || accountInfo.Token != tokenString {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
			return
		}

		c.Set("account_id", accountInfo.ID)
		c.Set("username", accountInfo.Username)
		c.Next()
	}
}

func extractBearerToken(authHeader string) (string, error) {
	before, after, found := strings.Cut(strings.TrimSpace(authHeader), " ")
	if !found || !strings.EqualFold(before, "Bearer") || strings.TrimSpace(after) == "" {
		return "", errors.New("invalid authorization header")
	}
	return strings.TrimSpace(after), nil
}

func GetAccountID(c *gin.Context) (uint, error) {
	value, exists := c.Get("account_id")
	if !exists {
		return 0, errors.New("account_id not found in gin.Context")
	}
	accountID, ok := value.(uint)
	if !ok {
		return 0, errors.New("account_id has invalid type in gin.Context")
	}
	return accountID, nil
}

func GetAccountUsername(c *gin.Context) (string, error) {
	value, exists := c.Get("username")
	if !exists {
		return "", errors.New("username not found in gin.Context")
	}
	username, ok := value.(string)
	if !ok {
		return "", errors.New("username has invalid type in gin.Context")
	}
	return username, nil
}
