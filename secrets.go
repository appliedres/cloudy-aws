package cloudyaws

import (
	"encoding/base64"
	"fmt"

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
	sec := cfg.(*AwsSecreatManager)
	if sec == nil {
		return nil, cloudy.InvalidConfigurationError
	}
	return sec
}

func (c *AwsSecretManagerFactory) ToConfig(config map[string]interface{}) (interface{}, error) {
	var found bool
	cfg := &AwsSecreatManager{}
	cfg.Region, found = cloudy.MapKeyStr(config, "Region", true)
	if !found {
		return nil, errors.New("Region required")
	}
	return cfg, nil
}

type AwsSecreatManager struct {
	Region string
}

func (a *AwsSecreatManager) SaveSecret(ctx context.Context, key string, secret string) error {
	region := a.Region

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(),
		aws.NewConfig().WithRegion(region))

	input := &secretsmanager.PutSecretValueInput{
		Name:     aws.String(secretName),
		SecretString: aws.String(secret),
	}

	_, err := svc.PutSecretValue(input)
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsSecreatManager) SaveSecretBinary(ctx context.Context, key string, secret []byte]) error {
	region := a.Region

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(),
		aws.NewConfig().WithRegion(region))

	input := &secretsmanager.PutSecretValueInput{
		Name:     aws.String(secretName),
		SecretBinary: secret,
	}

	_, err := svc.PutSecretValue(input)
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsSecreatManager) GetSecretBinary(ctx context.Context, key string) ([]byte, error) {
	_, data, err := a.getRawSecret(key)
	return data, err
}
func (a *AwsSecreatManager) GetSecret(key string) (ctx context.Context, string, error) {
	str, _, err := a.getRawSecret(key)
	return str, err
}

func (a *AwsSecreatManager) getRawSecret(key string) (string, []byte, error) {
	region := a.Region

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(),
		aws.NewConfig().WithRegion(region))
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
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
		return "", err
	}

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString, decodedBinarySecret string

	if result.SecretString != nil {
		secretString = *result.SecretString
		return secretString, nil, err
	}

	decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
	len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
	if err != nil {
		fmt.Println("Base64 Decode Error:", err)
		return "", err
	}
	return "", decodedBinarySecretBytes, err
}
