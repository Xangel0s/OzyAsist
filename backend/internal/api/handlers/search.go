package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/search"
)

func GlobalSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query param 'q' is required"})
		return
	}

	messages, chats, projects, memory, err := search.GlobalSearch(query, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to DTOs
	chatDTOs := make([]gin.H, 0, len(chats))
	for _, ch := range chats {
		chatDTOs = append(chatDTOs, gin.H{
			"id":    ch.ID,
			"title": ch.Title,
			"kind":  ch.Kind,
			"score": ch.Score,
		})
	}

	projectDTOs := make([]gin.H, 0, len(projects))
	for _, p := range projects {
		projectDTOs = append(projectDTOs, gin.H{
			"id":    p.ID,
			"name":  p.Title,
			"kind":  p.Kind,
			"score": p.Score,
		})
	}

	messageDTOs := make([]gin.H, 0, len(messages))
	for _, m := range messages {
		messageDTOs = append(messageDTOs, gin.H{
			"id":      m.ID,
			"snippet": m.Snippet,
			"score":   m.Score,
			"chat":    m.Title,
		})
	}

	memoryDTOs := make([]gin.H, 0, len(memory))
	for _, m := range memory {
		memoryDTOs = append(memoryDTOs, gin.H{
			"id":      m.ID,
			"text":    m.Snippet,
			"topic":   m.Title,
			"score":   m.Score,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messageDTOs,
		"chats":    chatDTOs,
		"projects": projectDTOs,
		"memory":   memoryDTOs,
	})
}
