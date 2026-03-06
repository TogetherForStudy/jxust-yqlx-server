package utils

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TokenClaims is the shared claim shape for access and refresh tokens.
type TokenClaims struct {
	UserID    uint   `json:"user_id"`
	Role      int8   `json:"role,omitempty"`
	TokenType string `json:"token_type"`
	JTI       string `json:"jti"`
	SID       string `json:"sid"`
	jwt.RegisteredClaims
}

type DeviceInfo struct {
	DeviceType string `json:"device_type"`
	ClientType string `json:"client_type"`
}

type AuthSession struct {
	UserID        uint   `json:"user_id"`
	RefreshJTI    string `json:"refresh_jti"`
	DeviceType    string `json:"device_type"`
	ClientType    string `json:"client_type"`
	IssuedAt      int64  `json:"issued_at"`
	LastRefreshAt int64  `json:"last_refresh_at"`
	ExpiresAt     int64  `json:"expires_at"`
}

type AuthBlockInfo struct {
	Type           string `json:"type"`
	Reason         string `json:"reason,omitempty"`
	OperatorUserID uint   `json:"operator_user_id,omitempty"`
	ExpiresAt      int64  `json:"expires_at"`
}

func GenerateAccessToken(userID uint, secret string, role int8, ttl time.Duration, sid string) (string, *TokenClaims, error) {
	return generateToken(userID, secret, role, ttl, sid, constant.AuthTokenTypeAccess)
}

func GenerateRefreshToken(userID uint, secret string, ttl time.Duration, sid string) (string, *TokenClaims, error) {
	return generateToken(userID, secret, 0, ttl, sid, constant.AuthTokenTypeRefresh)
}

func generateToken(userID uint, secret string, role int8, ttl time.Duration, sid, tokenType string) (string, *TokenClaims, error) {
	issuedAt := time.Now().UTC()
	claims := &TokenClaims{
		UserID:    userID,
		Role:      role,
		TokenType: tokenType,
		JTI:       uuid.NewString(),
		SID:       sid,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", nil, err
	}
	return tokenString, claims, nil
}

func ParseToken(tokenString, secret string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func NewSessionID() string {
	return uuid.NewString()
}

func ParseDeviceInfo(userAgent string) DeviceInfo {
	ua := strings.ToLower(userAgent)
	info := DeviceInfo{
		DeviceType: constant.AuthDeviceTypeUnknown,
		ClientType: constant.AuthClientTypeUnknown,
	}

	switch {
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ios") || strings.Contains(ua, "ipad"):
		info.DeviceType = constant.AuthDeviceTypeIOS
	case strings.Contains(ua, "android"):
		info.DeviceType = constant.AuthDeviceTypeAndroid
	case strings.Contains(ua, "windows"):
		info.DeviceType = constant.AuthDeviceTypeWindows
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macintosh"):
		info.DeviceType = constant.AuthDeviceTypeMac
	case strings.Contains(ua, "linux"):
		info.DeviceType = constant.AuthDeviceTypeLinux
	}

	if strings.Contains(ua, "miniprogram") || strings.Contains(ua, "mini program") || strings.Contains(ua, "micromessenger") {
		info.ClientType = constant.AuthClientTypeMiniProgram
	}

	return info
}

// GenerateJWT 生成JWT token
func GenerateJWT(userID uint, secret string, role int8) (string, error) {
	token, _, err := GenerateAccessToken(userID, secret, role, 7*24*time.Hour, NewSessionID())
	return token, err
}

// HashPassword 密码加密
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Pagination 分页参数
// Page 和 Size 由请求传入，Offset 由分页器推导。
type Pagination struct {
	Page   int `form:"page" json:"page"`
	Size   int `form:"size" json:"size"`
	Offset int `json:"-"`
}

// GetPagination 获取分页参数
func GetPagination(page, size int) *Pagination {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 10
	}

	return &Pagination{
		Page:   page,
		Size:   size,
		Offset: (page - 1) * size,
	}
}
