package jwt

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrTokenInvalid = errors.New("token is invalid or expired")
)

// Configurable TTL
const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

func getSecretKey() []byte {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		panic("JWT_SECRET_KEY is not set in environment")
	}
	return []byte(secret)
}

// GenerateAccessToken - faqat access token
func GenerateAccessToken(userID, role string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	claims["user_id"] = userID
	claims["role"] = role
	claims["typ"] = "access"
	claims["exp"] = time.Now().Add(AccessTokenTTL).Unix()
	claims["iat"] = time.Now().Unix()

	return token.SignedString(getSecretKey())
}

// GenerateRefreshToken - faqat refresh token
func GenerateRefreshToken(userID string) (string, string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	jti := fmt.Sprintf("%d", time.Now().UnixNano()) // unique ID

	claims["user_id"] = userID
	claims["typ"] = "refresh"
	claims["jti"] = jti
	claims["exp"] = time.Now().Add(RefreshTokenTTL).Unix()
	claims["iat"] = time.Now().Unix()

	signed, err := token.SignedString(getSecretKey())
	if err != nil {
		return "", "", err
	}
	return signed, jti, nil
}

// ParseToken - tokenni tekshiradi va claims qaytaradi
func ParseToken(tokenString string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return getSecretKey(), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		result := make(map[string]interface{})
		for k, v := range claims {
			result[k] = v
		}
		return result, nil
	}
	return nil, ErrTokenInvalid
}

// ExtractClaims - faqat kerakli maydonlarni oladi
func ExtractClaims(tokenString string) (map[string]interface{}, error) {
	parsedClaims, err := ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	if userID, ok := parsedClaims["user_id"]; ok {
		result["user_id"] = stringify(userID)
	}
	if role, ok := parsedClaims["role"]; ok {
		result["role"] = stringify(role)
	}
	if typ, ok := parsedClaims["typ"]; ok {
		result["typ"] = stringify(typ)
	}
	if jti, ok := parsedClaims["jti"]; ok {
		result["jti"] = stringify(jti)
	}
	return result, nil
}

func stringify(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
