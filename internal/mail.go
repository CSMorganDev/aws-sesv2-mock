package internal

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Destination struct {
	ToAddresses  []string `json:"ToAddresses"`
	CcAddresses  []string `json:"CcAddresses"`
	BccAddresses []string `json:"BccAddresses"`
}

type Content struct {
	Data    string `json:"Data"`
	CharSet string `json:"CharSet"`
}

type Body struct {
	Text Content `json:"Text"`
	Html Content `json:"Html"`
}

type ContentSubject struct {
	Data string `json:"Data"`
}

type EmailContent struct {
	Simple Message `json:"Simple"`
}

type Message struct {
	Body    Body    `json:"Body"`
	Subject Content `json:"Subject"`
}

type SendEmailRequest struct {
	Destination                 Destination  `json:"Destination"`
	Content                     EmailContent `json:"Content"`
	FromEmailAddress            string       `json:"FromEmailAddress"`
	FromEmailAddressIdentityArn string       `json:"FromEmailAddressIdentityArn"`
	ReplyToAddresses            []string     `json:"ReplyToAddresses"`
}

func isEmailInvalid(email string) bool {
	_, err := mail.ParseAddress(email)
	return err != nil
}

func SendEmail(request SendEmailRequest, c *gin.Context, dataDir string, logDir string) error {
	// Validate email addresses
	for _, address := range request.Destination.ToAddresses {
		if isEmailInvalid(address) {
			return errors.New("To-Address is invalid: " + address)
		}
	}

	for _, address := range request.Destination.CcAddresses {
		if isEmailInvalid(address) {
			return errors.New("CC-Address is invalid: " + address)
		}
	}

	for _, address := range request.Destination.BccAddresses {
		if isEmailInvalid(address) {
			return errors.New("BCC-Address is invalid: " + address)
		}
	}

	for _, address := range request.ReplyToAddresses {
		if isEmailInvalid(address) {
			return errors.New("Reply-To-Address is invalid: " + address)
		}
	}

	// Validation
	if !(request.FromEmailAddress != "" &&
		request.Content.Simple.Subject.Data != "" &&
		(request.Content.Simple.Body.Html.Data != "" || request.Content.Simple.Body.Text.Data != "") &&
		len(request.Destination.ToAddresses) > 0) {

		LogValidationErrors(&request)

		return errors.New("one or more required fields was not sent")
	}

	// Mkdir dataDir and logDir
	err := os.Mkdir(dataDir, 0755)
	if err != nil && os.IsNotExist(err) {
		return err
	}

	err = os.Mkdir(logDir, 0755)
	if err != nil && os.IsNotExist(err) {
		return err
	}

	// Write html data to logDir/body.html
	err = writeFileContent(filepath.Join(logDir, "body.html"), []byte(request.Content.Simple.Body.Html.Data))
	if err != nil {
		return err
	}

	// Write body to logDir/body.txt
	err = writeFileContent(filepath.Join(logDir, "body.txt"), []byte(request.Content.Simple.Body.Text.Data))
	if err != nil {
		return err
	}

	// Write headers to logDir/headers.txt
	headers := fmt.Sprintf("Subject: %s\nTo: %s\nCc: %s\nBcc: %s\nReply-To: %s\nFrom: %s\n",
		request.Content.Simple.Subject.Data,
		strings.Join(request.Destination.ToAddresses, ","),
		strings.Join(request.Destination.CcAddresses, ","),
		strings.Join(request.Destination.BccAddresses, ","),
		strings.Join(request.ReplyToAddresses, ","),
		request.FromEmailAddress,
	)
	err = writeFileContent(filepath.Join(logDir, "headers.txt"), []byte(headers))
	if err != nil {
		return err
	}

	// Generate a mock message ID
	messageId := fmt.Sprintf("mock-message-id-%d", time.Now().UnixNano())

	// Return JSON response (SESv2 format)
	c.JSON(http.StatusOK, gin.H{
		"MessageId": messageId,
	})

	return nil
}

func LogValidationErrors(request *SendEmailRequest) {
	// Check if ToAddresses is provided
	if len(request.Destination.ToAddresses) == 0 {
		logrus.Info("ToAddresses is not provided")
	}

	if request.FromEmailAddress == "" {
		logrus.Error("FromEmailAddress was not provided")
	}

	// Check if Subject is provided
	if request.Content.Simple.Subject.Data == "" {
		logrus.Error("Subject.Data was not provided")
	}

	// Check if Body.Html.Data or Body.Text.Data is provided
	if request.Content.Simple.Body.Html.Data == "" && request.Content.Simple.Body.Text.Data == "" {
		logrus.Error("Body.Html.Data or Body.Text.Data was not provided")
	}
}
