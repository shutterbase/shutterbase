package api_error

import "github.com/gin-gonic/gin"

type ApiError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type ErrorCode string

var (
	OK           = ApiError{Status: 200, Message: "OK"}
	BAD_REQUEST  = ApiError{Status: 400, Message: "Bad Request"}
	UNAUTHORIZED = ApiError{Status: 401, Message: "Unauthorized"}
	FORBIDDEN    = ApiError{Status: 403, Message: "Forbidden"}
	NOT_FOUND    = ApiError{Status: 404, Message: "Not Found"}
	INTERNAL     = ApiError{Status: 500, Message: "Internal Server Error"}
)

const (
	EMAIL_EXISTS ErrorCode = "EMAIL_EXISTS"
)

func (e ApiError) Send(c *gin.Context) {
	c.JSON(e.Status, gin.H{"message": e.Message})
}

func (e ApiError) SendWithCode(c *gin.Context, code ErrorCode) {
	c.JSON(e.Status, gin.H{"message": e.Message, "code": code})
}
