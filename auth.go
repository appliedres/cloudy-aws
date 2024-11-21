package cloudyaws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/appliedres/cloudy"
)

const DefaultRegion = "us-gov-east-1"

type AwsCredentials struct {
	Type            string // Can be any type of CredType*
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Location        string
}

const (
	CredTypeCli     = "cli"
	CredTypeDevCli  = "devcli"
	CredTypeSecret  = "secret"
	CredTypeCode    = "devicecode"
	CredTypeDefault = "default"
	CredTypeEnv     = "env"
	CredTypeBrowser = "browser"
	CredTypeManaged = "managed"
	CredTypeStatic  = "static"
	CredTypeIAMRole = "iam"
	CredTypeOther   = "other"
)

const (
	RegionPublic            = "public"
	RegionUSGovernment      = "usgovernment"
	RegionAzureUSGovernment = "azureusgoverment"
)

func fixRegionName(regionName string) string {
	regionNameFixed := strings.ToLower(regionName)
	regionNameFixed = strings.ReplaceAll(regionNameFixed, "-", "")
	regionNameFixed = strings.ReplaceAll(regionNameFixed, "_", "")
	return regionNameFixed
}

// func PolicyFromRegionString(regionName string) cloud.Configuration {
// 	regionNameFixed := fixRegionName(regionName)

// 	switch regionNameFixed {
// 	// Default to the government region
// 	case "":
// 		return cloud.AzureGovernment
// 	case RegionUSGovernment:
// 		return cloud.AzureGovernment
// 	case RegionAzureUSGovernment:
// 		return cloud.AzureGovernment
// 	case RegionPublic:
// 		return cloud.AzurePublic
// 	default:
// 		// Not sure WHAT to do with a custom.. Just assume it is the authority?
// 		// Needs to match "https://login.microsoftonline.com/"
// 		customRegion := regionName
// 		if !strings.HasPrefix(customRegion, "https://") {
// 			customRegion = fmt.Sprintf("https://%v", customRegion)
// 		}
// 		if !strings.HasSuffix(customRegion, "/") {
// 			customRegion = fmt.Sprintf("%v/", customRegion)
// 		}
// 		return cloud.Configuration{
// 			ActiveDirectoryAuthorityHost: customRegion,
// 			Services:                     map[cloud.ServiceName]cloud.ServiceConfiguration{},
// 		}
// 	}
// }

func NewAwsCredentials(awsCred *AwsCredentials) (aws.CredentialsProvider, error) {
	credType := strings.ToLower(awsCred.Type)
	if credType == "" {
		if awsCred.AccessKeyID != "" && awsCred.SecretAccessKey != "" {
			credType = CredTypeStatic
		} else {
			credType = CredTypeDefault
		}
	}

	switch credType {
	case CredTypeStatic:
		return credentials.NewStaticCredentialsProvider(
			awsCred.AccessKeyID,
			awsCred.SecretAccessKey,
			awsCred.SessionToken,
		), nil

	// case CredTypeEnv:
	// 	// Environment-based credentials provider
	// 	return aws.NewCredentialsCache(credentials.NewEnvironmentCredential()), nil

	// case CredTypeDefault:
	// 	// Default credential chain provider
	// 	cfg, err := config.LoadDefaultConfig(context.Background())
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to load default AWS config: %w", err)
	// 	}
	// 	return cfg.Credentials, nil

	default:
		return nil, fmt.Errorf("unknown credential type: %v", credType)
	}
}

// func GetAzureClientSecretCredential(azCfg AzureCredentials) (*azidentity.ClientSecretCredential, error) {

// 	cred, err := azidentity.NewClientSecretCredential(azCfg.TenantID, azCfg.ClientID, azCfg.ClientSecret,
// 		&azidentity.ClientSecretCredentialOptions{
// 			ClientOptions: policy.ClientOptions{
// 				Cloud: cloud.AzureGovernment,
// 			},
// 		})

// 	if err != nil {
// 		fmt.Printf("GetAzureCredentials Error authentication provider: %v\n", err)
// 		return nil, err
// 	}

// 	return cred, err
// }

func GetAzureCredentialsFromEnv(env *cloudy.Environment) AwsCredentials {
	// Check to see if there is already a set of credentials
	creds := env.GetCredential(AwsCredentialsKey)
	if creds != nil {
		return creds.(AwsCredentials)
	}
	credentials := AwsCredentials{
		Region:          env.Default("AWS_REGION", DefaultRegion),
		Type:            env.Default("AWS_CRED_TYPE", ""),
		AccessKeyID:     env.Default("AWS_ACCESS_KEY", ""),
		SecretAccessKey: env.Default("AWS_SECRET_KEY", ""),
		Location:        env.Default("AWS_LOCATION", ""),
	}
	return credentials
}
