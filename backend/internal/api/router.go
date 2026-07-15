package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/api/handlers"
	"github.com/ozyassist/backend/internal/api/middleware"
	"github.com/ozyassist/backend/internal/api/ws"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/providers"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	// Global middleware
	r.Use(middleware.CORS())
	r.Use(middleware.Auth())

	apiGroup := r.Group("/api")
	{
		// Auth
		apiGroup.POST("/auth/register", handlers.Register)
		apiGroup.POST("/auth/login", handlers.Login)
		apiGroup.PUT("/users/profile", handlers.UpdateProfile)

		// Chat
		apiGroup.GET("/chats", handlers.ListChats)
		apiGroup.POST("/chats", handlers.CreateChat)
		apiGroup.PATCH("/chats/:id", handlers.UpdateChat)
		apiGroup.GET("/chats/:id", handlers.GetChat)
		apiGroup.DELETE("/chats/:id", handlers.DeleteChat)
		apiGroup.POST("/chats/:id/messages", handlers.SendMessage)
		apiGroup.PATCH("/chats/:id/messages/:messageId/feedback", handlers.UpdateMessageFeedback)

		// Projects
		apiGroup.GET("/projects", handlers.ListProjects)
		apiGroup.POST("/projects", handlers.CreateProject)
		apiGroup.GET("/projects/:id", handlers.GetProject)
		apiGroup.PUT("/projects/:id", handlers.UpdateProject)
		apiGroup.DELETE("/projects/:id", handlers.DeleteProject)
		apiGroup.POST("/projects/:id/index", handlers.IndexProject)
		apiGroup.GET("/projects/:id/tree", handlers.GetProjectTree)
		apiGroup.GET("/projects/:id/graph/*filepath", handlers.GetProjectGraph)
		apiGroup.GET("/projects/:id/fullgraph", handlers.GetAllProjectGraph)
		apiGroup.POST("/projects/:id/upload-files", handlers.UploadProjectFiles)

		// Skills
		apiGroup.GET("/skills", handlers.ListSkills)
		apiGroup.POST("/skills", handlers.CreateSkill)
		apiGroup.POST("/skills/execute", handlers.ExecuteSkill)
		apiGroup.DELETE("/skills/:id", handlers.DeleteSkill)

		// Connectors
		apiGroup.GET("/connectors", handlers.ListConnectors)
		apiGroup.POST("/connectors", handlers.CreateConnector)
		apiGroup.DELETE("/connectors/:id", handlers.DeleteConnector)

		// Memory
		apiGroup.POST("/memory/import", handlers.ImportMemory)
		apiGroup.GET("/memory/search", handlers.SearchMemory)

		// Global search
		apiGroup.GET("/search", handlers.GlobalSearch)

		// Models
		apiGroup.GET("/models", handlers.ListModels)
		apiGroup.POST("/models/select", handlers.SelectModel)

		// Files
		apiGroup.POST("/files/upload", handlers.UploadFile)
		apiGroup.GET("/files/:id", handlers.GetFile)
		apiGroup.GET("/files/:id/download", handlers.GetFile)

		// Agent
		apiGroup.POST("/agent/tasks", handlers.CreateTask)
		apiGroup.GET("/agent/tasks/:id", handlers.GetTask)
		apiGroup.POST("/agent/tasks/:id/cancel", handlers.CancelTask)
		apiGroup.POST("/agent/tasks/:id/confirm", handlers.ConfirmAction)

		// Sidebar
		apiGroup.POST("/sidebar/observe", handlers.Observe)
		apiGroup.POST("/sidebar/command", handlers.SidebarCommand)

		// Settings
		apiGroup.GET("/settings", handlers.GetSettings)
		apiGroup.PUT("/settings", handlers.UpdateSettings)

		// Onboarding
		apiGroup.POST("/onboarding/analyze-memory", handlers.AnalyzeMemory)
	}

	// WebSocket
	r.GET("/ws", ws.HandleWebSocket)

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "0.1.0"})
	})

	// Debug
	r.GET("/debug", func(c *gin.Context) {
		var userCount int
		var chatCount int
		db.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
		db.DB.QueryRow("SELECT COUNT(*) FROM chats").Scan(&chatCount)
		c.JSON(200, gin.H{
			"users":        userCount,
			"chats":        chatCount,
			"default_user": db.DefaultUserID(),
			"providers":   providers.Available(),
		})
	})

	return r
}
