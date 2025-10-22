package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AWSErrorResponse represents the standard AWS error response format for SESv2
type AWSErrorResponse struct {
	Type    string `json:"__type"`
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

func handler(c *gin.Context) {
	// Read the request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, AWSErrorResponse{
			Type:    "InternalServiceException",
			Code:    "InternalServiceException",
			Message: "Failed to read request body",
		})
		return
	}

	// Build dateDir
	dateTime := time.Now().Format("2006-01-02-15-04-05.000Z")
	dateDir := Config.OutputDir + "/" + dateTime[:10]
	logDir := dateDir + "/" + dateTime[11:22] + "-log"

	var request SendEmailRequest
	err = json.Unmarshal(bodyBytes, &request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, AWSErrorResponse{
			Type:    "InvalidParameterValue",
			Code:    "InvalidParameterValue",
			Message: "Failed to parse JSON request: " + err.Error(),
		})
		return
	}

	mailErr := SendEmail(request, c, dateDir, logDir)
	if mailErr != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, AWSErrorResponse{
			Type:    "InvalidParameterValue",
			Code:    "InvalidParameterValue",
			Message: mailErr.Error(),
		})
		return
	}
}

func StartServer() {
	// Read environment variables
	ReadConfigFromEnv()
	logrus.Info("Starting mock server under port ", Config.Port)

	// Endpoints
	r := gin.Default()
	r.POST("/v2/email/outbound-emails", handler) // SESv2 SendEmail endpoint

	// Run
	err := r.Run(":" + strconv.Itoa(Config.Port))
	if err != nil {
		panic(err)
	}
}
