package cloudyaws

// TODO: implement back off and retry for creating/getting deleted secrets, as they can take up to 2 hours to delete

import (
	"context"
	"fmt"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/secrets"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

const AwsSecretManagerID = "aws"

func init() {
	secrets.SecretProviders.Register(AwsSecretManagerID, &AwsSecretManagerFactory{})
}

type AwsSecretManagerFactory struct{}

type AwsSecretManagerConfig struct {
	AwsCredentials
}

func (c *AwsSecretManagerFactory) Create(cfg interface{}) (secrets.SecretProvider, error) {
	fmt.Println("AWS SecretManager: Create")

	sec := cfg.(*AwsSecretManager)
	if sec == nil {
		return nil, cloudy.ErrInvalidConfiguration
	}
	return NewSecretManager(context.Background(), sec.AwsCredentials)
}

func (c *AwsSecretManagerFactory) FromEnv(env *cloudy.Environment) (interface{}, error) {
	fmt.Println("AWS SecretManager: FromEnv")

	cfg := &AwsSecretManager{}
	cfg.AwsCredentials = GetAwsCredentialsFromEnv(env)
	return cfg, nil
}

type AwsSecretManager struct {
	AwsCredentials
}


func NewSecretManager(ctx context.Context, creds AwsCredentials) (*AwsSecretManager, error) {
	cloudy.Info(ctx, "AWS SecretManager: NewSecretManager")
	return &AwsSecretManager{
		AwsCredentials: creds,
	}, nil
}

func (a *AwsSecretManager) ListAll(ctx context.Context) ([]string, error) {	
	cloudy.Info(ctx, "AWS SecretManager: ListAll")

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(), aws.NewConfig().WithRegion(a.AwsCredentials.Region))

	// Set the input parameters for listing secrets
	input := &secretsmanager.ListSecretsInput{}

	// Call the ListSecrets API
	result, err := svc.ListSecrets(input)
	if err != nil {
		// Handle errors
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case secretsmanager.ErrCodeInvalidNextTokenException:
				cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidNextTokenException, aerr.Error())
			case secretsmanager.ErrCodeInvalidParameterException:
				cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidParameterException, aerr.Error())
			case secretsmanager.ErrCodeResourceNotFoundException:
				cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			default:
				cloudy.Info(ctx, aerr.Error())
			}
		} else {
			cloudy.Info(ctx, err.Error())
		}
		return nil, err
	}

	// Print the list of secrets
	cloudy.Info(ctx, "Secrets:")
	var secretNames []string
	for _, secret := range result.SecretList {
		cloudy.Info(ctx, *secret.Name)
		secretNames = append(secretNames, *secret.Name)
	}
	return secretNames, err
}


func (a *AwsSecretManager) SaveSecret(ctx context.Context, key string, secret string) error {
	cloudy.Info(ctx, "saving raw secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)

	err := a.createSecretRaw(ctx, key, secret)
	if err != nil {
		err = a.putSecretRaw(ctx, key, secret)
		if err != nil {
			return err
		}
		cloudy.Info(ctx, "successfully put raw secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)
		return nil
	}

	cloudy.Info(ctx, "successfully created raw secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)
	return nil
}

func (a *AwsSecretManager) SaveSecretBinary(ctx context.Context, key string, secret []byte) error {
	cloudy.Info(ctx, "saving binary secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)

	err := a.createSecretBinary(ctx, key, secret)
	if err != nil {
		err = a.putSecretBinary(ctx, key, secret)
		if err != nil {
			return err
		}
		cloudy.Info(ctx, "successfully put binary secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)
		return nil
	}

	cloudy.Info(ctx, "successfully created binary secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)
	return nil
}

func (a *AwsSecretManager) GetSecret(ctx context.Context, key string) (string, error) {
	cloudy.Info(ctx, "getting raw secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)
	str, _, err := a.getRawSecret(ctx, key)
	return str, err
}

func (a *AwsSecretManager) GetSecretBinary(ctx context.Context, key string) ([]byte, error) {
	cloudy.Info(ctx, "getting binary secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)
	_, data, err := a.getRawSecret(ctx, key)
	return data, err
}

func (a *AwsSecretManager) DeleteSecret(ctx context.Context, key string) error {
	cloudy.Info(ctx, "deleting secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)
	svc := secretsmanager.New(session.New(), aws.NewConfig().WithRegion(a.AwsCredentials.Region))

	_, err := svc.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId: aws.String(key),
		ForceDeleteWithoutRecovery: aws.Bool(true),

	})

	return err
}

func (a *AwsSecretManager) putSecretRaw(ctx context.Context, key string, secret string) error {
	cloudy.Info(ctx, "putting raw secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(), aws.NewConfig().WithRegion(a.AwsCredentials.Region))
	
	input := &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(key),
		SecretString: aws.String(secret),
	}

	_, err := svc.PutSecretValue(input)
    if err != nil {
        // Handle errors using awserr.
        if aerr, ok := err.(awserr.Error); ok {
            switch aerr.Code() {
            case secretsmanager.ErrCodeInvalidParameterException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidParameterException, aerr.Error())
            case secretsmanager.ErrCodeInvalidRequestException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidRequestException, aerr.Error())
            case secretsmanager.ErrCodeLimitExceededException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeLimitExceededException, aerr.Error())
            case secretsmanager.ErrCodeEncryptionFailure:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeEncryptionFailure, aerr.Error())
            case secretsmanager.ErrCodeResourceExistsException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeResourceExistsException, aerr.Error())
            case secretsmanager.ErrCodeMalformedPolicyDocumentException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeMalformedPolicyDocumentException, aerr.Error())
            case secretsmanager.ErrCodeInternalServiceError:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInternalServiceError, aerr.Error())
            default:
                cloudy.Info(ctx, aerr.Error())
            }
        } else {
            cloudy.Info(ctx, err.Error())
        }
        return err
    }

	return nil
}

func (a *AwsSecretManager) putSecretBinary(ctx context.Context, key string, secret []byte) error {
	cloudy.Info(ctx, "putting raw secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(), aws.NewConfig().WithRegion(a.AwsCredentials.Region))

	input := &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(key),
		SecretBinary: secret,
	}

	_, err := svc.PutSecretValue(input)
    if err != nil {
        // Handle errors using awserr.
        if aerr, ok := err.(awserr.Error); ok {
            switch aerr.Code() {
            case secretsmanager.ErrCodeInvalidParameterException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidParameterException, aerr.Error())
            case secretsmanager.ErrCodeInvalidRequestException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidRequestException, aerr.Error())
            case secretsmanager.ErrCodeLimitExceededException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeLimitExceededException, aerr.Error())
            case secretsmanager.ErrCodeEncryptionFailure:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeEncryptionFailure, aerr.Error())
            case secretsmanager.ErrCodeResourceExistsException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeResourceExistsException, aerr.Error())
            case secretsmanager.ErrCodeMalformedPolicyDocumentException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeMalformedPolicyDocumentException, aerr.Error())
            case secretsmanager.ErrCodeInternalServiceError:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInternalServiceError, aerr.Error())
            default:
                cloudy.Info(ctx, aerr.Error())
            }
        } else {
            cloudy.Info(ctx, err.Error())
        }
        return err
    }

	return nil
}

func (a *AwsSecretManager) createSecretRaw(ctx context.Context, key string, secret string) error {
	cloudy.Info(ctx, "creating raw secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)
	
	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(), aws.NewConfig().WithRegion(a.AwsCredentials.Region))

	input := &secretsmanager.CreateSecretInput{
		Name:     aws.String(key),
		SecretString: aws.String(secret),
	}

	_, err := svc.CreateSecret(input)
    if err != nil {
        // Handle errors using awserr.
        if aerr, ok := err.(awserr.Error); ok {
            switch aerr.Code() {
            case secretsmanager.ErrCodeInvalidParameterException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidParameterException, aerr.Error())
            case secretsmanager.ErrCodeInvalidRequestException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidRequestException, aerr.Error())
            case secretsmanager.ErrCodeLimitExceededException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeLimitExceededException, aerr.Error())
            case secretsmanager.ErrCodeEncryptionFailure:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeEncryptionFailure, aerr.Error())
            case secretsmanager.ErrCodeResourceExistsException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeResourceExistsException, aerr.Error())
            case secretsmanager.ErrCodeMalformedPolicyDocumentException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeMalformedPolicyDocumentException, aerr.Error())
            case secretsmanager.ErrCodeInternalServiceError:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInternalServiceError, aerr.Error())
            default:
                cloudy.Info(ctx, aerr.Error())
            }
        } else {
            cloudy.Info(ctx, err.Error())
        }
        return err
    }

	return nil
}

func (a *AwsSecretManager) createSecretBinary(ctx context.Context, key string, secret []byte) error {
	cloudy.Info(ctx, "creating raw secret with key [%s] in region [%s]", key, a.AwsCredentials.Region)

	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(), aws.NewConfig().WithRegion(a.AwsCredentials.Region))

	input := &secretsmanager.CreateSecretInput{
		Name:     aws.String(key),
		SecretBinary: secret,
	}

	_, err := svc.CreateSecret(input)
    if err != nil {
        // Handle errors using awserr.
        if aerr, ok := err.(awserr.Error); ok {
            switch aerr.Code() {
            case secretsmanager.ErrCodeInvalidParameterException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidParameterException, aerr.Error())
            case secretsmanager.ErrCodeInvalidRequestException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidRequestException, aerr.Error())
            case secretsmanager.ErrCodeLimitExceededException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeLimitExceededException, aerr.Error())
            case secretsmanager.ErrCodeEncryptionFailure:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeEncryptionFailure, aerr.Error())
            case secretsmanager.ErrCodeResourceExistsException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeResourceExistsException, aerr.Error())
            case secretsmanager.ErrCodeMalformedPolicyDocumentException:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeMalformedPolicyDocumentException, aerr.Error())
            case secretsmanager.ErrCodeInternalServiceError:
                cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInternalServiceError, aerr.Error())
            default:
                cloudy.Info(ctx, aerr.Error())
            }
        } else {
            cloudy.Info(ctx, err.Error())
        }
        return err
    }

	return nil
}

func (a *AwsSecretManager) getRawSecret(ctx context.Context, key string) (string, []byte, error) {
	//Create a Secrets Manager client
	svc := secretsmanager.New(session.New(), aws.NewConfig().WithRegion(a.AwsCredentials.Region))

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
				cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeDecryptionFailure, aerr.Error())

			case secretsmanager.ErrCodeInternalServiceError:
				// An error occurred on the server side.
				cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInternalServiceError, aerr.Error())

			case secretsmanager.ErrCodeInvalidParameterException:
				// You provided an invalid value for a parameter.
				cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidParameterException, aerr.Error())

			case secretsmanager.ErrCodeInvalidRequestException:
				// You provided a parameter value that is not valid for the current state of the resource.
				cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeInvalidRequestException, aerr.Error())

			case secretsmanager.ErrCodeResourceNotFoundException:
				// We can't find the resource that you asked for.
				cloudy.Info(ctx, "Exc:%s, Error:%s", secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			default:
                cloudy.Info(ctx, aerr.Error())
            }
			
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cloudy.Info(ctx, err.Error())
		}
		return "", nil, err
	}

	if result.SecretString != nil {
		return *result.SecretString, nil, err
	}

	if result.SecretBinary != nil {
		return "", result.SecretBinary, err
	}

	return "", nil, cloudy.Error(ctx, "could not find SecretString or SecretBinary")
}
