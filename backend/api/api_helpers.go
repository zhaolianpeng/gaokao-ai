package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gaokao-ai/backend/model"
)

func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}

func bindQuery(c *gin.Context, req any) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, model.ErrorResponse{Error: message})
}

func abortWithError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, model.ErrorResponse{Error: message})
}

func writeItems[T any](c *gin.Context, items []T) {
	c.JSON(http.StatusOK, model.ItemsResponse[T]{Items: items})
}
