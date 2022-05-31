package cloudyaws

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"fmt"
)

type Dynamo[T any] struct {
	Sess   *session.Session
	Client *dynamodb.DynamoDB
	Table  string
}

func NewDynamo[T any](tableName string) (*Dynamo[T], error) {
	// Initialize a session that the SDK will use to load
	// credentials from the shared credentials file ~/.aws/credentials
	// and region from the shared configuration file ~/.aws/config.
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	return &Dynamo[T]{
		Sess:   sess,
		Client: svc,
		Table:  tableName,
	}, nil
}

func (d *Dynamo[T]) Save(item *T) error {
	// Load the item
	itemMap, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		fmt.Println("Got error marshalling new movie item:")
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      itemMap,
		TableName: aws.String(d.Table),
	}

	_, err = d.Client.PutItem(input)
	if err != nil {
		fmt.Println("Got error calling PutItem:")
		return err
	}

	return nil
}

func (d *Dynamo[T]) Read(key string, attribute string) (*T, error) {
	var out T
	result, err := d.Client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(d.Table),
		Key: map[string]*dynamodb.AttributeValue{
			key: {S: aws.String(attribute)},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		msg := "Could not find '" + attribute + "'"
		return nil, errors.New(msg)
	}

	err = dynamodbattribute.UnmarshalMap(result.Item, &out)
	return &out, err
}
