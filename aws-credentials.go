package cloudyaws

import (
	"fmt"

	"github.com/appliedres/cloudy"
)

type AwsCredentials struct {
	Region     		string
	AccessKeyID     string
	SecretAccessKey string
}

func init() {
	cloudy.CredentialSources[AwsCredentialsKey] = &AwsCredentialLoader{}
}

const AwsCredentialsKey = "aws"

type AwsCredentialLoader struct{}

func (loader *AwsCredentialLoader) ReadFromEnv(env *cloudy.Environment) interface{} {
	fmt.Println("AWS Credentials: ReadFromEnv")
	region := env.Get("AWS_REGION")
	if region == "" {
		region = "us-gov-west-1"
	}
	accessKeyId := env.Get("AWS_ACCESS_KEY_ID")
	secretAccessKey := env.Get("AWS_SECRET_ACCESS_KEY")

	if accessKeyId == "" || secretAccessKey == "" {
		return nil
	}

	return AwsCredentials{
		Region:       		region,
		AccessKeyID:    	accessKeyId,
		SecretAccessKey:    secretAccessKey,
	}

}

func GetAwsCredentialsFromEnv(env *cloudy.Environment) AwsCredentials {
	fmt.Println("AWS Credentials: GetAwsCredentialsFromEnv")

	// Check to see if there is already a set of credentials
	creds := env.GetCredential(AwsCredentialsKey)
	if creds != nil {
		return creds.(AwsCredentials)
	}

	return AwsCredentials{
		Region:       env.Force("AWS_REGION"),
		AccessKeyID:     env.Force("AWS_ACCESS_KEY_ID"),
		SecretAccessKey:     env.Force("AWS_SECRET_ACCESS_KEY"),
	}
}
