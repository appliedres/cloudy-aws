package cloudyaws

import (
	"context"
	"errors"
	"fmt"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/models"
)

func init() {
	cloudy.GroupProviders.Register(AwsCognito, &CognitoGroupManagerFactory{})
}

/*

The Cognito User Manager is the implementation of the `UserManager` interface from cloudy.

"Amazon Cognito lets you add user sign-up, sign-in, and access control to your
web and mobile apps quickly and easily. Amazon Cognito scales to millions of users and
supports sign-in with social identity providers, such as Apple, Facebook, Google, and
Amazon, and enterprise identity providers via SAML 2.0 and OpenID Connect."
-- https://aws.amazon.com/cognito/

AWS Cognito has the concept of user pools. Each pool is created with a set of user attributes that
cannot be changed after the fact. It is recommended that you keep a small amount of information in the
cognito profile and that you use a secondary Account object to gather the remaining information.

The standard attributes supported by cognito are: address, nickname, birthdate, phone number, email,
picture, family name, preferred username, gender, profile, given name, zoneinfo, locale, updated at,
middle name, website, and name. Since these choices are IMMUTABLE after pool creation you should choose
carefully. You can also add custom attributes.

If AWS Cognito is used as the ONLY user management approach then select the fields that you need.

*/

type CognitoGroupManagerFactory struct{}

func (c *CognitoGroupManagerFactory) Create(cfg interface{}) (cloudy.GroupManager, error) {
	cogCfg := cfg.(*CognitoConfig)
	if cogCfg == nil {
		return nil, cloudy.InvalidConfigurationError
	}
	return NewCognitoUserManager(cogCfg)
}

func (c *CognitoGroupManagerFactory) ToConfig(config map[string]interface{}) (interface{}, error) {
	var found bool
	cogCfg := &CognitoConfig{}
	cogCfg.PoolID, found = cloudy.MapKeyStr(config, "PoolID", true)
	if !found {
		return nil, errors.New("PoolID required")
	}

	cogCfg.Region, found = cloudy.MapKeyStr(config, "Region", true)
	if !found {
		return nil, errors.New("Region required")
	}

	cogCfg.Region, found = cloudy.MapKeyStr(config, "UserAttributes", true)
	if !found {
		return nil, errors.New("Region required")
	}

	return cogCfg, nil
}

type CognitoGroupManager struct {
	Client *Cognito
	Config *CognitoConfig
}

func NewCognitoGroupManager(cfg *CognitoConfig) (*CognitoGroupManager, error) {
	if cfg.PoolID == "" {
		return nil, fmt.Errorf("invalid PoolID")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("invalid Region")
	}
	if len(cfg.UserAttributes) == 0 {
		return nil, fmt.Errorf("no user attributes specified")
	}

	cog := NewCognito(cfg.Region, cfg.PoolID)
	cog.UserAttributes = cfg.UserAttributes
	return &CognitoGroupManager{
		Client: cog,
		Config: cfg,
	}, nil
}

func (c *CognitoUserManager) ListGroups(ctx context.Context, uid string) ([]*models.Group, error) {
	return c.Client.ListGroups()
}

func (c *CognitoUserManager) GetUserGroups(ctx context.Context, uid string) ([]*models.Group, error) {
	groups, err := c.Client.GetUserGroups(uid)
	return groups, err
}

func (c *CognitoUserManager) NewGroup(ctx context.Context, grp *models.Group) (*models.Group, error) {
	err := c.Client.CreateGroup(grp)
	return grp, err
}

func (c *CognitoUserManager) AddMembers(ctx context.Context, groupId string, userIds []string) error {
	merr := cloudy.MultiError()

	for _, uid := range userIds {
		err := c.Client.AddUserToGroup(groupId, uid)
		if err != nil {
			merr.Append(fmt.Errorf("error adding %v to %v, %v", uid, groupId, err))
		}
	}

	return merr.AsErr()
}

func (c *CognitoUserManager) RemoveMembers(ctx context.Context, groupId string, userIds []string) error {
	merr := cloudy.MultiError()

	for _, uid := range userIds {
		err := c.Client.RemoveUserFromGroup(groupId, uid)
		if err != nil {
			merr.Append(fmt.Errorf("error removing %v to %v, %v", uid, groupId, err))
		}
	}

	return merr.AsErr()
}

func (c *CognitoUserManager) UpdateGroup(ctx context.Context, grp *models.Group) (bool, error) {
	return false, nil

}

func (c *CognitoUserManager) GetGroupMembers(ctx context.Context, grpId string) ([]*models.User, error) {
	users, err := c.Client.ListAllUsersInGroup(grpId)
	return users, err
}
