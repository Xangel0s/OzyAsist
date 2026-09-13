package api

import (
	"os"
	"path/filepath"
	"strings"

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

	// Disable internal Gin redirects that drop CORS headers
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// Apply CORS to 404 and 405 handlers explicitly
	r.NoRoute(middleware.CORS(), func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "Ruta no encontrada"})
	})
	r.NoMethod(middleware.CORS(), func(c *gin.Context) {
		c.JSON(405, gin.H{"error": "Método no permitido"})
	})

	apiGroup := r.Group("/api")
	{
		// Auth & Profiles
		apiGroup.GET("/auth/profiles", handlers.ListProfiles)
		apiGroup.POST("/auth/profiles", handlers.CreateProfile)
		apiGroup.POST("/auth/register", handlers.Register)
		apiGroup.POST("/auth/login", handlers.Login)
		apiGroup.POST("/auth/pin/verify", handlers.VerifyPin)
		apiGroup.POST("/auth/pin/update", handlers.UpdatePin)
		apiGroup.DELETE("/auth/profiles/:id", handlers.DeleteProfile)
		apiGroup.PUT("/users/profile", handlers.UpdateProfile)

		// Chat
		apiGroup.GET("/chats", handlers.ListChats)
		apiGroup.POST("/chats", handlers.CreateChat)
		apiGroup.PATCH("/chats/:id", handlers.UpdateChat)
		apiGroup.GET("/chats/:id", handlers.GetChat)
		apiGroup.DELETE("/chats/:id", handlers.DeleteChat)
		apiGroup.POST("/chats/:id/messages", handlers.SendMessage)
		apiGroup.PATCH("/chats/:id/messages/:messageId/feedback", handlers.UpdateMessageFeedback)

		// MCP Connectors
		apiGroup.GET("/mcp", handlers.ListMCPConnectors)
		apiGroup.POST("/mcp", handlers.CreateMCPConnector)
		apiGroup.DELETE("/mcp/:id", handlers.DeleteMCPConnector)
		apiGroup.POST("/mcp/:id/reconnect", handlers.ReconnectMCPConnector)

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
		apiGroup.GET("/projects/:id/file", handlers.GetFileContent)
		apiGroup.POST("/projects/:id/upload-files", handlers.UploadProjectFiles)

		// Git Operations
		apiGroup.GET("/projects/:id/git/status", handlers.GetGitStatus)
		apiGroup.POST("/projects/:id/git/init", handlers.InitGit)
		apiGroup.POST("/projects/:id/git/stage", handlers.StageGit)
		apiGroup.POST("/projects/:id/git/unstage", handlers.UnstageGit)
		apiGroup.POST("/projects/:id/git/commit", handlers.CommitGit)
		apiGroup.POST("/projects/:id/git/sync", handlers.SyncGit)
		apiGroup.POST("/projects/:id/git/generate-msg", handlers.GenerateGitCommitMessage)

		// Terminal Operations
		apiGroup.POST("/projects/:id/terminal/exec", handlers.ExecuteTerminalCommand)

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

		// Models & Providers
		apiGroup.GET("/models", handlers.ListModels)
		apiGroup.GET("/models/available", handlers.GetAvailableModels)
		apiGroup.POST("/models/select", handlers.SelectModel)
		apiGroup.POST("/providers/test", handlers.TestProviderConnection)

		// Files
		apiGroup.POST("/files/upload", handlers.UploadFile)
		apiGroup.GET("/files/:id", handlers.GetFile)
		apiGroup.GET("/files/:id/download", handlers.GetFile)

		// Agent
		apiGroup.POST("/agent/tasks", handlers.CreateTask)
		apiGroup.GET("/agent/tasks/:id", handlers.GetTask)
		apiGroup.POST("/agent/tasks/:id/cancel", handlers.CancelTask)
		apiGroup.POST("/agent/tasks/:id/confirm", handlers.ConfirmAction)
		apiGroup.POST("/tasks/:id/authorize", handlers.AuthorizeTask)
		apiGroup.POST("/tasks/:id/cancel", handlers.CancelTask)
		apiGroup.POST("/tasks/emergency-kill", handlers.EmergencyKill)
		apiGroup.POST("/agent/tasks/emergency-kill", handlers.EmergencyKill)

		// Sidebar
		apiGroup.POST("/sidebar/observe", handlers.Observe)
		apiGroup.POST("/sidebar/command", handlers.SidebarCommand)

		// Security EDR & Audit Trail
		apiGroup.GET("/security/audit", handlers.GetAuditTrail)
		apiGroup.GET("/security/snapshots/:taskId", handlers.GetTaskSnapshots)
		apiGroup.POST("/security/undo/:taskId", handlers.UndoTask)
		apiGroup.POST("/security/watchdog/kill", handlers.KillSuspectProcess)

		// Settings
		apiGroup.GET("/settings", handlers.GetSettings)
		apiGroup.PUT("/settings", handlers.UpdateSettings)

		// Audio & Native Hardware Voice
		apiGroup.GET("/audio/devices", handlers.GetAudioDevicesHandler)
		apiGroup.GET("/audio/status", handlers.GetAudioStatusHandler)
		apiGroup.POST("/audio/wakeword/test", handlers.TestWakeWordHandler)

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

	// Servir frontend embebido o desde directorio físico
	distPath := "../frontend/dist"
	if _, err := os.Stat(distPath); err != nil {
		cwd, _ := os.Getwd()
		distPath = filepath.Join(cwd, "frontend", "dist")
	}
	if _, err := os.Stat(distPath); err == nil {
		r.NoRoute(func(c *gin.Context) {
			p := c.Request.URL.Path
			if strings.HasPrefix(p, "/api") || strings.HasPrefix(p, "/ws") {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			filePath := filepath.Join(distPath, filepath.Clean(p))
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				c.File(filePath)
				return
			}
			c.File(filepath.Join(distPath, "index.html"))
		})
	} else {
		RegisterSPA(r)
	}

	return r
}
