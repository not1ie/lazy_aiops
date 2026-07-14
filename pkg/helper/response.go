package helper

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents a unified HTTP response structure.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success outputs a successful JSON response.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Error outputs an error JSON response.
// If the error message contains sensitive database details, it shields them.
func Error(c *gin.Context, httpStatus int, code int, message string, rawErr error) {
	if rawErr != nil {
		log.Printf("[API_ERROR] Code: %d, Message: %s, RawErr: %v", code, message, rawErr)
	}
	
	// Shield system/database internal error details from the client
	displayMsg := message
	if displayMsg == "" {
		displayMsg = "系统内部错误"
	}
	
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: displayMsg,
	})
}
