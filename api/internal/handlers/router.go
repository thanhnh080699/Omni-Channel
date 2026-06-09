package handlers

import (
	"net/http"
	"sync"
	"time"

	"omni-channel/backend/internal/auth"
	"omni-channel/backend/internal/config"
	"omni-channel/backend/internal/database"
	"omni-channel/backend/internal/middleware"
	"omni-channel/backend/internal/queue"
	"omni-channel/backend/internal/rbac"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	cfg     config.Config
	db      *database.Mongo
	tokens  *auth.TokenService
	rbac    *rbac.Checker
	queue   queue.Publisher
	qrCache *qrCache
}

type qrCache struct {
	mu      sync.Mutex
	entries map[string]qrCacheEntry
}

type qrCacheEntry struct {
	QR        string
	Status    string
	UpdatedAt time.Time
	ExpiresAt time.Time
}

func NewRouter(cfg config.Config, db *database.Mongo, tokens *auth.TokenService, publisher queue.Publisher) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(cors())

	h := &Handler{
		cfg:     cfg,
		db:      db,
		tokens:  tokens,
		rbac:    rbac.NewChecker(db),
		queue:   publisher,
		qrCache: &qrCache{entries: map[string]qrCacheEntry{}},
	}

	router.GET("/health", h.health)
	router.POST("/api/auth/login", h.login)
	router.POST("/webhooks/:channel/:accountId", h.receiveWebhook)
	router.POST("/internal/whatsapp/message-status", h.whatsAppMessageStatus)

	api := router.Group("/api")
	api.Use(middleware.Auth(db, tokens))
	api.GET("/auth/profile", h.profile)
	api.PATCH("/auth/profile", h.updateProfile)
	api.POST("/auth/change-password", h.changePassword)
	api.POST("/auth/logout", h.logout)
	api.POST("/auth/refresh", h.refresh)
	api.GET("/notifications/chat", h.chatNotifications)

	admin := api.Group("/admin")
	admin.Use(h.requirePermission("admin:manage"))
	admin.GET("/users", h.listUsers)
	admin.POST("/users", h.createUser)
	admin.PATCH("/users/:userId", h.updateUser)
	admin.DELETE("/users/:userId", h.deleteUser)
	admin.GET("/roles", h.listRoles)
	admin.POST("/roles", h.createRole)
	admin.PATCH("/roles/:roleId", h.updateRole)
	admin.POST("/users/:userId/roles", h.assignRoles)
	admin.POST("/users/:userId/teams", h.assignTeams)
	admin.GET("/permissions/matrix", h.permissionMatrix)
	admin.GET("/teams", h.listTeams)
	admin.POST("/teams", h.createTeam)
	admin.PATCH("/teams/:teamId", h.updateTeam)
	admin.DELETE("/teams/:teamId", h.deleteTeam)
	admin.GET("/channels", h.listChannels)
	admin.GET("/channel-accounts", h.listChannelAccounts)
	admin.POST("/channel-accounts", h.createChannelAccount)
	admin.PATCH("/channel-accounts/:accountId", h.updateChannelAccount)
	admin.POST("/channel-accounts/:accountId/enable", h.enableChannelAccount)
	admin.POST("/channel-accounts/:accountId/disable", h.disableChannelAccount)
	admin.GET("/channel-accounts/:accountId/health", h.channelAccountHealth)
	admin.GET("/audit-logs", h.listAuditLogs)

	channelAdmin := api.Group("/channel-admin")
	channelAdmin.Use(h.requireAnyPermission("admin:manage", "channel:manage"))
	channelAdmin.GET("/channels", h.listChannels)
	channelAdmin.GET("/teams", h.listTeams)
	channelAdmin.GET("/users", h.listUsers)
	channelAdmin.GET("/channel-accounts", h.listChannelAccounts)
	channelAdmin.POST("/channel-accounts", h.createChannelAccount)
	channelAdmin.PATCH("/channel-accounts/:accountId", h.updateChannelAccount)
	channelAdmin.POST("/channel-accounts/:accountId/enable", h.enableChannelAccount)
	channelAdmin.POST("/channel-accounts/:accountId/disable", h.disableChannelAccount)
	channelAdmin.GET("/channel-accounts/:accountId/health", h.channelAccountHealth)
	channelAdmin.GET("/channel-accounts/:accountId/whatsapp/session", h.whatsAppSession)
	channelAdmin.POST("/channel-accounts/:accountId/whatsapp/connect", h.whatsAppConnect)
	channelAdmin.POST("/channel-accounts/:accountId/whatsapp/disconnect", h.whatsAppDisconnect)
	channelAdmin.POST("/channel-accounts/:accountId/whatsapp/reset-session", h.whatsAppResetSession)
	channelAdmin.POST("/channel-accounts/:accountId/whatsapp/resync", h.whatsAppResync)

	api.GET("/conversations/my", h.listMyConversations)
	api.GET("/conversations/team", h.listTeamConversations)
	api.GET("/conversations/trash", h.listTrashConversations)
	api.POST("/conversations", h.createConversation)
	api.GET("/conversations/:conversationId", h.getConversation)
	api.DELETE("/conversations/:conversationId", h.deleteConversation)
	api.POST("/conversations/:conversationId/restore", h.restoreConversation)
	api.POST("/conversations/:conversationId/assign", h.assignConversation)
	api.POST("/conversations/:conversationId/transfer", h.assignConversation)
	api.PATCH("/conversations/:conversationId/tags", h.updateConversationTags)
	api.GET("/conversations/:conversationId/typing", h.conversationTyping)
	api.GET("/conversations/:conversationId/messages", h.listMessages)
	api.POST("/conversations/:conversationId/messages", h.sendMessage)
	api.POST("/conversations/:conversationId/read", h.markConversationRead)
	api.POST("/messages/:messageId/retry", h.retryMessage)
	api.GET("/chat/channels", h.listChannels)
	api.GET("/chat/channel-accounts", h.listChatChannelAccounts)
	api.GET("/chat/channel-accounts/:accountId/check-phone", h.checkChannelAccountPhone)
	api.GET("/chat/channel-accounts/:accountId/avatar", h.getChannelAccountAvatar)

	return router
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP", "service": "omni-channel-api"})
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
