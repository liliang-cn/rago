package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Auth creates an authentication middleware based on the specified type
func Auth(authType, secret string) gin.HandlerFunc {
	switch authType {
	case "bearer":
		return BearerAuth(secret)
	case "basic":
		return BasicAuth(secret)
	case "api_key":
		return APIKeyAuth(secret)
	default:
		return BearerAuth(secret)
	}
}

// BearerAuth implements JWT bearer token authentication
func BearerAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate JWT
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Extract claims and add to context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("user_id", claims["sub"])
			c.Set("claims", claims)
		}

		c.Next()
	}
}

// BasicAuth implements HTTP basic authentication
func BasicAuth(credentials string) gin.HandlerFunc {
	// Parse credentials in format "username:password"
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		panic("invalid basic auth credentials format")
	}

	expectedUsername := parts[0]
	expectedPassword := parts[1]

	return gin.BasicAuth(gin.Accounts{
		expectedUsername: expectedPassword,
	})
}

// APIKeyAuth implements API key authentication
func APIKeyAuth(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check header first
		key := c.GetHeader("X-API-Key")
		
		// Fall back to query parameter
		if key == "" {
			key = c.Query("api_key")
		}

		if key == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		if key != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth creates an optional authentication middleware
func OptionalAuth(authType, secret string) gin.HandlerFunc {
	authMiddleware := Auth(authType, secret)

	return func(c *gin.Context) {
		// Try to authenticate but don't block if it fails
		authMiddleware(c)
		
		// Check if authentication was successful
		if c.IsAborted() {
			// Clear the abort flag and continue
			c.Set("authenticated", false)
			c.Abort() // Reset abort flag
			c.Next()
		} else {
			c.Set("authenticated", true)
		}
	}
}