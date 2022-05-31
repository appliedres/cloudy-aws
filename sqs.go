package cloudyaws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Queue simple wrapper for SQS actions
type Queue struct {
	Client *sqs.SQS
}

//NewQueue creates a new Queue wrapper
func NewQueue() *Queue {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := sqs.New(sess)

	return &Queue{
		Client: svc,
	}
}

//Recieve get messages off the topic queue
func (q *Queue) Recieve(topic string) ([]*sqs.Message, error) {
	out, err := q.Client.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(topic),
		VisibilityTimeout:   aws.Int64(60),
		WaitTimeSeconds:     aws.Int64(0),
		MaxNumberOfMessages: aws.Int64(10),
	})

	if err != nil {
		return nil, err
	}

	return out.Messages, nil
}

//Send sends a message
func (q *Queue) Send(topic string, message string) (*string, error) {
	out, err := q.Client.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(message),
		QueueUrl:    aws.String(topic),
	})

	return out.MessageId, err
}

//Delete removes messages from the topic
func (q *Queue) Delete(topic string, handle *string) error {
	_, err := q.Client.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(topic),
		ReceiptHandle: handle,
	})

	// fmt.Printf("DELETEING %v from %v. %v \n", handle, topic, err)
	return err
}
