package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())
	
	router.GET("/test", func(c *gin.Context) {
		requestID, exists := c.Get("request_id")
		assert.True(t, exists)
		assert.NotEmpty(t, requestID)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	// Test with no request ID header
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))

	// Test with existing request ID header
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-id")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-request-id", w.Header().Get("X-Request-ID"))
}

func TestBearerAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret"
	
	// Create a valid JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	validToken, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid format",
			authHeader:     "InvalidFormat",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(BearerAuth(secret))
			
			router.GET("/protected", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				c.JSON(http.StatusOK, gin.H{"user_id": userID})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			router.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestAPIKeyAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	apiKey := "test-api-key"

	tests := []struct {
		name           string
		headerKey      string
		queryKey       string
		expectedStatus int
	}{
		{
			name:           "valid header key",
			headerKey:      apiKey,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid query key",
			queryKey:       apiKey,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid key",
			headerKey:      "wrong-key",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(APIKeyAuth(apiKey))
			
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			w := httptest.NewRecorder()
			path := "/protected"
			if tt.queryKey != "" {
				path += "?api_key=" + tt.queryKey
			}
			req, _ := http.NewRequest("GET", path, nil)
			if tt.headerKey != "" {
				req.Header.Set("X-API-Key", tt.headerKey)
			}

			router.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimit(60, 5)) // 60 requests per minute, burst of 5
	
	router.GET("/limited", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Should allow burst of requests
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/limited", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// Next request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/limited", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestIPRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(IPRateLimit(3, 100*time.Millisecond))
	
	router.GET("/limited", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Should allow 3 requests
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/limited", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 4th request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/limited", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should allow request again
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/limited", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := zerolog.Nop()
	
	router := gin.New()
	router.Use(RequestID())
	router.Use(Logger(logger))
	
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?param=value", nil)
	req.Header.Set("User-Agent", "test-agent")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())
	router.Use(Recovery())
	
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal server error")
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := DefaultCORSConfig()
	
	router := gin.New()
	router.Use(CORS(config))
	
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test preflight request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))

	// Test actual request
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestStrictCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	
	router := gin.New()
	router.Use(StrictCORS(allowedOrigins))
	
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test allowed origin
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))

	// Test disallowed origin
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestOptionalAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	apiKey := "test-key"
	
	router := gin.New()
	router.Use(OptionalAuth("api_key", apiKey))
	
	router.GET("/test", func(c *gin.Context) {
		authenticated, _ := c.Get("authenticated")
		c.JSON(http.StatusOK, gin.H{"authenticated": authenticated})
	})

	// Test with valid auth
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", apiKey)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"authenticated":true`)

	// Test without auth - should still work
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"authenticated":false`)
}