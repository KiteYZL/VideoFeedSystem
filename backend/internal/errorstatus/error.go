package errorstatus

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrPasswordEmpty = errors.New("password can't be empty")

// 将错误映射至http状态码，无法识别的错误默认返回500
func ErrorToStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	switch {
	case errors.Is(err, context.Canceled):
		return http.StatusRequestTimeout
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return http.StatusUnauthorized
	case errors.Is(err, jwt.ErrTokenMalformed),
		errors.Is(err, jwt.ErrTokenSignatureInvalid),
		errors.Is(err, jwt.ErrTokenExpired),
		errors.Is(err, jwt.ErrTokenNotValidYet),
		errors.Is(err, jwt.ErrTokenInvalidClaims):
		return http.StatusUnauthorized
	}

	var validationErr validator.ValidationErrors
	if errors.As(err, &validationErr) {
		return http.StatusBadRequest
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return http.StatusConflict
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "validation failed"):
		return http.StatusBadRequest
	case errors.Is(err, ErrPasswordEmpty):
		return http.StatusBadRequest
	case strings.Contains(msg, "password can't be empty"):
		return http.StatusBadRequest
	case strings.Contains(msg, "invalid authorization header"):
		return http.StatusUnauthorized
	case strings.Contains(msg, "account_id not found"):
		return http.StatusUnauthorized
	case strings.Contains(msg, "account_id has invalid type"):
		return http.StatusUnauthorized
	case strings.Contains(msg, "username not found"):
		return http.StatusUnauthorized
	case strings.Contains(msg, "username has invalid type"):
		return http.StatusUnauthorized
	case strings.Contains(msg, "token has been revoked"):
		return http.StatusUnauthorized
	case strings.Contains(msg, "not found"):
		return http.StatusNotFound
	case strings.Contains(msg, "duplicate entry"):
		return http.StatusConflict
	case strings.Contains(msg, "unauthorized"):
		return http.StatusUnauthorized
	}

	return http.StatusInternalServerError
}
