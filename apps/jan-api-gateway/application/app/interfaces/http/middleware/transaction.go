package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

func TransactionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tx := database.DB.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()
		ctxWithTx := transaction.WithTx(c.Request.Context(), tx)
		c.Request = c.Request.WithContext(ctxWithTx)
		c.Next()

		if c.IsAborted() {
			tx.Rollback()
			return
		}

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code: "a5a38af2-1605-4f58-a89c-fa3ff390d4db",
			})
			return
		}
	}
}
