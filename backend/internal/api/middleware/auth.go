package middleware

import (
	"github.com/gin-gonic/gin"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Modo local sin auth — always pass
		// En modo servidor, validar JWT aquí
		c.Next()
	}
}
