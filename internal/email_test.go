package internal

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

var _ = Describe("SES", func() {
	Context("sending an email", func() {
		var (
			svc *sesv2.Client
		)

		BeforeSuite(func() {
			// Start the local server
			go func() {
				StartServer()
			}()

			// Set up a new SES session
			sesConfig, err := constructAWSConfiguration("http://localhost:8081")
			Expect(err).NotTo(HaveOccurred())

			// Create a new SES client
			svc = sesv2.NewFromConfig(sesConfig)
		})

		It("should send an email successfully", func() {
			// Set up the email message
			input := &sesv2.SendEmailInput{
				Destination: &types.Destination{
					ToAddresses: []string{
						"recipient@example.com",
					},
				},
				Content: &types.EmailContent{
					Simple: &types.Message{
						Body: &types.Body{
							Html: &types.Content{
								Data: aws.String("<p>Hello, world!</p>"),
							},
						},
						Subject: &types.Content{
							Data: aws.String("Test email"),
						},
					},
				},
				FromEmailAddress:            aws.String("sender@example.com"),
				FromEmailAddressIdentityArn: aws.String("arn:aws:ses:eu-central-1:123456789012:identity/sender@example.com"),
			}

			// Send the email
			result, err := svc.SendEmail(context.TODO(), input)
			Expect(err).NotTo(HaveOccurred())

			// Check the response
			Expect(result).NotTo(BeNil())
			Expect(result.MessageId).NotTo(BeNil())
			Expect(*result.MessageId).NotTo(BeEmpty())

			fmt.Println("Email sent successfully!")
		})

		It("should fail to send an email if the recipient address is invalid", func() {
			// Set up the email message with an invalid recipient address
			input := &sesv2.SendEmailInput{
				Destination: &types.Destination{
					ToAddresses: []string{
						"invalid_email.com",
					},
				},
				Content: &types.EmailContent{
					Simple: &types.Message{
						Body: &types.Body{
							Html: &types.Content{
								Data: aws.String("<p>Hello, world!</p>"),
							},
						},
						Subject: &types.Content{
							Data: aws.String("Test email"),
						},
					},
				},
				FromEmailAddress:            aws.String("sender@example.com"),
				FromEmailAddressIdentityArn: aws.String("arn:aws:ses:eu-central-1:123456789012:identity/sender@example.com"),
			}

			// Send the email
			_, err := svc.SendEmail(context.TODO(), input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
	})
})

func TestEmailSendingCompliance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "")
}

func constructAWSConfiguration(endpoint string) (cfg aws.Config, err error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if endpoint != "" {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           endpoint,
				SigningRegion: "eu-central-1",
			}, nil
		}

		// returning EndpointNotFoundError will allow the service to fallback to it's default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	// Create an AWS configuration
	configArray := []func(options *config.LoadOptions) error{
		config.WithRegion("eu-central-1"),
		config.WithDefaultRegion("eu-central-1"),
		config.WithEndpointResolverWithOptions(customResolver),
	}

	// For local development we need to use anonymous credentials
	configArray = append(configArray, config.WithCredentialsProvider(aws.AnonymousCredentials{}))

	return config.LoadDefaultConfig(
		context.Background(),
		configArray...,
	)
}
