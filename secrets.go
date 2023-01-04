package cloudyaws

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/secrets"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

const AwsSecreatManagerID = "aws-secretmanager"

func init() {
	secrets.SecretProviders.Register(AwsSecreatManagerID, &AwsSecretManagerFactory{})
}

type AwsSecretManagerFactory struct{}

func (c *AwsSecretManagerFactory) Create(cfg interface{}) (secrets.SecretProvider, error) {
	sec := cfg.(*AwsSecretManager)
	if sec == nil {
		return nil, cloudy.ErrInvalidConfiguration
	}
	return sec, nil
}

func (c *AwsSecretManagerFactory) FromEnv(env *cloudy.Environment) (interface{}, error) {
	var found bool
	cfg := &AwsSecretManager{}
	cfg.Region, found = cloudy.MapKeyStr(config, "Region", true)
	if !found {
		return nil, errors.New("Region required")
	}
	return cfg, nil
}

type AwsSecretManager struct {
	Region string
}

func (a *AwsSecretManager) SaveSecret(ctx context.Context, key string, secret string) error {
	region := a.Region

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(),
		aws.NewConfig().WithRegion(region))

	input := &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(key),
		SecretString: aws.String(secret),
	}

	_, err := svc.PutSecretValue(input)
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsSecretManager) SaveSecretBinary(ctx context.Context, key string, secret []byte) error {
	region := a.Region

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(),
		aws.NewConfig().WithRegion(region))

	input := &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(key),
		SecretBinary: secret,
	}

	_, err := svc.PutSecretValue(input)
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsSecretManager) GetSecretBinary(ctx context.Context, key string) ([]byte, error) {
	_, data, err := a.getRawSecret(key)
	return data, err
}
func (a *AwsSecretManager) GetSecret(ctx context.Context, key string) (string, error) {
	str, _, err := a.getRawSecret(key)
	return str, err
}

func (a *AwsSecretManager) DeleteSecret(ctx context.Context, key string) error {
	svc := secretsmanager.New(session.New(), aws.NewConfig().WithRegion(a.Region))

	_, err := svc.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId: aws.String(key),
	})

	return err
}

func (a *AwsSecretManager) getRawSecret(key string) (string, []byte, error) {
	region := a.Region

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(),
		aws.NewConfig().WithRegion(region))
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(key),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	// In this sample we only handle the specific exceptions for the 'GetSecretValue' API.
	// See https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html

	result, err := svc.GetSecretValue(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case secretsmanager.ErrCodeDecryptionFailure:
				// Secrets Manager can't decrypt the protected secret text using the provided KMS key.
				fmt.Println(secretsmanager.ErrCodeDecryptionFailure, aerr.Error())

			case secretsmanager.ErrCodeInternalServiceError:
				// An error occurred on the server side.
				fmt.Println(secretsmanager.ErrCodeInternalServiceError, aerr.Error())

			case secretsmanager.ErrCodeInvalidParameterException:
				// You provided an invalid value for a parameter.
				fmt.Println(secretsmanager.ErrCodeInvalidParameterException, aerr.Error())

			case secretsmanager.ErrCodeInvalidRequestException:
				// You provided a parameter value that is not valid for the current state of the resource.
				fmt.Println(secretsmanager.ErrCodeInvalidRequestException, aerr.Error())

			case secretsmanager.ErrCodeResourceNotFoundException:
				// We can't find the resource that you asked for.
				fmt.Println(secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return "", nil, err
	}

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString string

	if result.SecretString != nil {
		secretString = *result.SecretString
		return secretString, nil, err
	}

	decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
	_, err = base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
	if err != nil {
		fmt.Println("Base64 Decode Error:", err)
		return "", nil, err
	}
	return "", decodedBinarySecretBytes, err
}
