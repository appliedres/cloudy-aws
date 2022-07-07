package cloudyaws

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/appliedres/cloudy"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"gopkg.in/gomail.v2"
)

func init() {
	cloudy.EmailerProviders.Register(AwsCognito, &SESEmailerFactory{})
}

type SESEmailerFactory struct{}

func (ses *SESEmailerFactory) Create(cfg interface{}) (cloudy.Emailer, error) {
	cogCfg := cfg.(*CognitoConfig)
	if cogCfg == nil {
		return nil, cloudy.ErrInvalidConfiguration
	}
	return NewSESEmailer()
}

func (ses *SESEmailerFactory) FromEnv(env *cloudy.SegmentedEnvironment) (interface{}, error) {
	return nil, nil
}

type SESEmailerConfig struct{}

// The SES Emailer is the AWS Simple Email Serivice implementation of the cloudy `Emailer` interface.

// Amazon Simple Email Service (SES) is a cost-effective, flexible, and scalable email service that enables
// developers to send mail from within any application. You can configure Amazon SES quickly to support
// several email use cases, including transactional, marketing, or mass email communications. Amazon SES's
// flexible IP deployment and email authentication options help drive higher deliverability and protect sender
// reputation, while sending analytics measure the impact of each email. With Amazon SES, you can send email
// securely, globally, and at scale. -- https://aws.amazon.com/ses/
//
// ## Usage Notes
// You need to make sure that you only send "from" your approved email addresses. Also, this mailer includes
// some capablities to check your email quota. Under the covers the send raw email method is being called to
// support attachements. https://docs.aws.amazon.com/ses/latest/APIReference/API_SendRawEmail.html
type SESEmailer struct {
	Client *ses.SES
}

type EmailQuota struct {
	Max24HourSend   *float64 `type:"double"`
	MaxSendRate     *float64 `type:"double"`
	SentLast24Hours *float64 `type:"double"`
	Remaining       uint64
}

func NewSESEmailer() (*SESEmailer, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	svc := ses.New(sess)

	return &SESEmailer{
		Client: svc,
	}, nil
}

func (m *SESEmailer) Send(ctx context.Context, msg *gomail.Message) error {
	var dest []*string

	for _, addr := range msg.GetHeader("to") {
		if len(strings.TrimSpace(addr)) > 0 {
			dest = append(dest, aws.String(addr))
		}
	}
	for _, addr := range msg.GetHeader("cc") {
		if len(strings.TrimSpace(addr)) > 0 {
			dest = append(dest, aws.String(addr))
		}
	}
	for _, addr := range msg.GetHeader("bcc") {
		if len(strings.TrimSpace(addr)) > 0 {
			dest = append(dest, aws.String(addr))
		}
	}
	from := msg.GetHeader("from")

	if len(dest) == 0 {
		return errors.New("No Destination (to, cc, bcc) addresses")
	}
	if len(from) != 1 {
		return errors.New("invalid from")
	}

	var emailRaw bytes.Buffer
	msg.WriteTo(&emailRaw)

	message := ses.RawMessage{Data: emailRaw.Bytes()}

	input := &ses.SendRawEmailInput{
		Source:       aws.String(from[0]),
		Destinations: dest,
		RawMessage:   &message,
	}

	_, err := m.Client.SendRawEmail(input)

	if err != nil {
		return err
	}

	// return *result.MessageId, nil
	return nil
}

func (m *SESEmailer) PauseSending() {
	m.Client.UpdateAccountSendingEnabled(&ses.UpdateAccountSendingEnabledInput{
		Enabled: aws.Bool(false),
	})

	fmt.Printf("Email Sending Paused")
}

func (m *SESEmailer) ResumeSending() {
	m.Client.UpdateAccountSendingEnabled(&ses.UpdateAccountSendingEnabledInput{
		Enabled: aws.Bool(true),
	})

	fmt.Printf("Email Sending Resumes")
}

//DecodeSendError turns the error into a status string
func (m *SESEmailer) DecodeSendError(err error) string {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case ses.ErrCodeMessageRejected:
			return fmt.Sprintf("%v, %v", ses.ErrCodeMessageRejected, aerr.Error())
		case ses.ErrCodeMailFromDomainNotVerifiedException:
			return fmt.Sprintf("%v, %v", ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
		case ses.ErrCodeConfigurationSetDoesNotExistException:
			return fmt.Sprintf("%v, %v", ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
		case ses.ErrCodeConfigurationSetSendingPausedException:
			return fmt.Sprintf("%v, %v", ses.ErrCodeConfigurationSetSendingPausedException, aerr.Error())
		case ses.ErrCodeAccountSendingPausedException:
			return fmt.Sprintf("%v, %v", ses.ErrCodeAccountSendingPausedException, aerr.Error())
		default:
			return fmt.Sprintf("%v", aerr.Error())
		}
	} else {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		return fmt.Sprintf("%v", err.Error())
	}
}

//Quota gets the quota for sending
func (m *SESEmailer) Quota() (*EmailQuota, error) {

	resp, err := m.Client.GetSendQuota(&ses.GetSendQuotaInput{})
	if err != nil {
		return nil, err
	}

	remaining := *resp.Max24HourSend - *resp.SentLast24Hours

	q := &EmailQuota{
		Max24HourSend:   resp.Max24HourSend,
		MaxSendRate:     resp.MaxSendRate,
		SentLast24Hours: resp.SentLast24Hours,
		Remaining:       uint64(remaining),
	}
	return q, nil
}
