package middleware

import (
	"net/http"
	"strings"

	"omni-channel/backend/internal/auth"
	"omni-channel/backend/internal/database"
	"omni-channel/backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

const CurrentUserKey = "current_user"

func Auth(db *database.Mongo, tokens *auth.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(raw, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := tokens.Parse(strings.TrimPrefix(raw, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		var user models.User
		if err := db.C("users").FindOne(c.Request.Context(), bson.M{"_id": claims.UserID, "status": "active"}).Decode(&user); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user is not active"})
			return
		}
		c.Set(CurrentUserKey, user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (models.User, bool) {
	value, ok := c.Get(CurrentUserKey)
	if !ok {
		return models.User{}, false
	}
	user, ok := value.(models.User)
	return user, ok
}
