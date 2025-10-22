package internal

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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
	Action                      string       `json:"Action"`
	Destination                 Destination  `json:"Destination"`
	Content                     EmailContent `json:"EmailContent"`
	FromEmailAddress            string       `json:"FromEmailAddress"`
	FromEmailAddressIdentityArn string       `json:"FromEmailAddressIdentityArn"`
	ReplyToAddresses            []string     `json:"ReplyToAddresses"`
}

func deserializeSendEmailRequest(reqBody string) (*SendEmailRequest, error) {
	queryValues, err := url.ParseQuery(reqBody)
	if err != nil {
		return nil, err
	}

	toAddresses := []string{queryValues.Get("Destination.ToAddresses.member.1")}

	// Then, initialize the struct fields using the map values
	sendEmailRequest := SendEmailRequest{
		Action: queryValues.Get("Action"),
		Destination: Destination{
			ToAddresses: toAddresses,
		},
		Content: EmailContent{
			Simple: Message{
				Body: Body{
					Html: Content{
						Data: queryValues.Get("Content.Simple.Body.Html.Data"),
					},
				},
				Subject: Content{
					Data: queryValues.Get("Content.Simple.Subject.Data"),
				},
			},
		},
		FromEmailAddress:            queryValues.Get("FromEmailAddress"),
		FromEmailAddressIdentityArn: queryValues.Get("FromEmailAddressIdentityArn"),
	}

	for _, address := range toAddresses {
		if isEmailInvalid(address) {
			return nil, errors.New("To-Address is invalid: " + address)
		}
	}

	// Optional fields
	if ccAddresses, ok := queryValues["Destination.CcAddresses.member.1"]; ok {
		sendEmailRequest.Destination.CcAddresses = ccAddresses
		for _, address := range ccAddresses {
			if isEmailInvalid(address) {
				return nil, errors.New("CC-Address is invalid: " + address)
			}
		}
	}

	if bccAddresses, ok := queryValues["Destination.BccAddresses.member.1"]; ok {
		sendEmailRequest.Destination.BccAddresses = bccAddresses
		for _, address := range bccAddresses {
			if isEmailInvalid(address) {
				return nil, errors.New("BCC-Address is invalid: " + address)
			}
		}
	}

	if replyToAddresses, ok := queryValues["ReplyToAddresses.member.1"]; ok {
		sendEmailRequest.ReplyToAddresses = replyToAddresses
		for _, address := range replyToAddresses {
			if isEmailInvalid(address) {
				return nil, errors.New("Reply-To-Address is invalid: " + address)
			}
		}
	}

	return &sendEmailRequest, nil
}

func isEmailInvalid(email string) bool {
	_, err := mail.ParseAddress(email)
	return err != nil
}

func SendEmail(bodyString string, c *gin.Context, dataDir string, logDir string) error {
	request, err := deserializeSendEmailRequest(bodyString)

	if err != nil {
		return err
	}

	// Validation
	if !(request.FromEmailAddress != "" &&
		request.Content.Simple.Subject.Data != "" &&
		(request.Content.Simple.Body.Html.Data != "" || request.Content.Simple.Body.Text.Data != "") &&
		len(request.Destination.ToAddresses) > 0) {

		LogValidationErrors(request)

		return errors.New("one or more required fields was not sent")
	}

	// Mkdir dataDir and logDir
	err = os.Mkdir(dataDir, 0755)
	if err != nil && os.IsNotExist(err) {
		return err
	}

	err = os.Mkdir(logDir, 0755)
	if err != nil && os.IsNotExist(err) {
		return err
	}

	// Write html data to dataDir/body.html
	err = writeFileContent(filepath.Join(logDir, "body.html"), []byte(request.Content.Simple.Body.Html.Data))
	if err != nil {
		return err
	}

	// Write body to dataDir/body.txt
	err = writeFileContent(filepath.Join(logDir, "body.txt"), []byte(request.Content.Simple.Body.Text.Data))
	if err != nil {
		return err
	}

	// Write headers to dataDir/headers.txt
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

	// Read file from templates/success.txt
	successTemplate, err := os.ReadFile("../assets/templates/success.xml")
	if err != nil {
		logrus.Error("Cannot open template success file: ", err)
		return err
	}

	// Replace {{message}} with absolute path of the body.html
	successMessage := strings.Replace(string(successTemplate), "{{message}}", filepath.Join(dataDir, "body.html"), -1)

	// Respond with the content & 200
	c.String(http.StatusOK, successMessage)

	return nil
}

func LogValidationErrors(request *SendEmailRequest) {
	// Check if ToAddresses is provided
	if len(request.Destination.ToAddresses) < 0 {
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

func SendRawEmail(c *gin.Context, dateDir string, logFilePath string) {
	// TODO

	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Not implemented",
	})
}
