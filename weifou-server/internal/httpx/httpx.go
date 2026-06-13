package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIError 业务错误，带错误码与 HTTP 状态。
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string { return e.Message }

func NewError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

// 常用构造
func BadRequest(code, msg string) *APIError   { return NewError(http.StatusBadRequest, code, msg) }
func Unauthorized(code, msg string) *APIError { return NewError(http.StatusUnauthorized, code, msg) }
func Forbidden(code, msg string) *APIError    { return NewError(http.StatusForbidden, code, msg) }
func NotFound(code, msg string) *APIError     { return NewError(http.StatusNotFound, code, msg) }
func TooManyRequests(code, msg string) *APIError {
	return NewError(http.StatusTooManyRequests, code, msg)
}
func Internal(code, msg string) *APIError {
	return NewError(http.StatusInternalServerError, code, msg)
}

// OK 统一成功响应：{ success: true, data }
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// Fail 统一失败响应：{ success: false, code, message }
func Fail(c *gin.Context, err error) {
	if apiErr, ok := err.(*APIError); ok {
		c.JSON(apiErr.Status, gin.H{
			"success": false,
			"code":    apiErr.Code,
			"message": apiErr.Message,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"code":    "INTERNAL_ERROR",
		"message": "服务器开小差了，请稍后再试",
	})
}

// Handle 包装 handler，统一处理错误返回。
func Handle(fn func(c *gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := fn(c); err != nil {
			Fail(c, err)
		}
	}
}
