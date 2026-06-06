package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"meditour/cdn/config"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Rate Limiting per IP
var (
	limiters = make(map[string]*rate.Limiter)
	mu       sync.Mutex
)

func ResetLimiters() {
	mu.Lock()
	defer mu.Unlock()
	limiters = make(map[string]*rate.Limiter)
}

func getLimiter(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	l, exists := limiters[ip]
	if !exists {
		l = rate.NewLimiter(rate.Limit(config.AppConfig.RateLimitRPS), config.AppConfig.RateLimitBurst)
		limiters[ip] = l
	}

	return l
}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		limiter := getLimiter(c.ClientIP())
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// Signed URL Middleware
func SignatureMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only check signature if enabled in config
		if !config.AppConfig.RequireSignature {
			c.Next()
			return
		}

		// Example URL with signature: /uploads/abc.jpg?sig=xyz&exp=123
		signature := c.Query("sig")
		expiry := c.Query("exp")

		if signature == "" || expiry == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Signature or expiry missing"})
			c.Abort()
			return
		}

		expUnix, err := strconv.ParseInt(expiry, 10, 64)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Invalid expiry"})
			c.Abort()
			return
		}

		if time.Now().Unix() > expUnix {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: URL expired"})
			c.Abort()
			return
		}

		// Verify HMAC
		path := c.Param("filepath")
		// Correct way: sign the combination of path + query params (except sig)
		// For simplicity/standard, we follow a specific pattern:
		// dataToSign = path + "?" + allParamsSortedButExcludingSig

		// Simplest: hmac(path + expiry + secret)
		data := fmt.Sprintf("%s%s", path, expiry)
		h := hmac.New(sha256.New, []byte(config.AppConfig.SignatureKey))
		h.Write([]byte(data))
		expectedSignature := hex.EncodeToString(h.Sum(nil))

		if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Invalid signature"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-KEY")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey != config.AppConfig.ApiKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid API Key"})
			c.Abort()
			return
		}
		c.Next()
	}
}
