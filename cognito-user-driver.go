package cloudyaws

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"strings"

// 	"github.com/appliedres/cloudy"
// 	"github.com/appliedres/cloudy/models"
// )

// const AwsCognito = "cognito"

// const Attr_address = "address"
// const Attr_nickname = "nickname"
// const Attr_birthdate = "birthdate"
// const Attr_phoneNumber = "phone number"
// const Attr_email = "email"
// const Attr_picture = "picture"
// const Attr_familyName = "family name"
// const Attr_preferredUsername = "preferred username"
// const Attr_gender = "gender"
// const Attr_profile = "profile"
// const Attr_givenName = "given name"
// const Attr_zoneinfo = "zoneinfo"
// const Attr_locale = "locale"
// const Attr_updatedAt = "updated at"
// const Attr_middleName = "middle name"
// const Attr_website = "website"
// const Attr_name = "name"

// var DefaultUserAttributes = []string{
// 	Attr_email, Attr_name, Attr_givenName, Attr_familyName,
// }

// func init() {
// 	cloudy.UserProviders.Register(AwsCognito, &CognitoUserManagerFactory{})
// }

// /*
// The Cognito User Manager is the implementation of the `UserManager` interface from cloudy.

// "Amazon Cognito lets you add user sign-up, sign-in, and access control to your
// web and mobile apps quickly and easily. Amazon Cognito scales to millions of users and
// supports sign-in with social identity providers, such as Apple, Facebook, Google, and
// Amazon, and enterprise identity providers via SAML 2.0 and OpenID Connect."
// -- https://aws.amazon.com/cognito/

// AWS Cognito has the concept of user pools. Each pool is created with a set of user attributes that
// cannot be changed after the fact. It is recommended that you keep a small amount of information in the
// cognito profile and that you use a secondary Account object to gather the remaining information.

// The standard attributes supported by cognito are: address, nickname, birthdate, phone number, email,
// picture, family name, preferred username, gender, profile, given name, zoneinfo, locale, updated at,
// middle name, website, and name. Since these choices are IMMUTABLE after pool creation you should choose
// carefully. You can also add custom attributes.

// If AWS Cognito is used as the ONLY user management approach then select the fields that you need.
// */

// type CognitoUserManagerFactory struct{}

// func (c *CognitoUserManagerFactory) Create(cfg interface{}) (cloudy.UserManager, error) {
// 	cogCfg := cfg.(*CognitoConfig)
// 	if cogCfg == nil {
// 		return nil, cloudy.ErrInvalidConfiguration
// 	}
// 	return NewCognitoUserManager(cogCfg)
// }

// func (c *CognitoUserManagerFactory) FromEnv(env *cloudy.Environment) (interface{}, error) {
// 	var found bool
// 	cogCfg := &CognitoConfig{}
// 	cogCfg.PoolID, found = cloudy.MapKeyStr(config, "PoolID", true)
// 	if !found {
// 		return nil, errors.New("PoolID required")
// 	}

// 	cogCfg.Region, found = cloudy.MapKeyStr(config, "Region", true)
// 	if !found {
// 		return nil, errors.New("Region required")
// 	}

// 	userAttributes, found := cloudy.MapKeyStr(config, "UserAttributes", true)
// 	if !found {
// 		cogCfg.UserAttributes = DefaultUserAttributes
// 	} else {
// 		attrs := strings.Split(userAttributes, ",")
// 		for i, attr := range attrs {
// 			attrs[i] = strings.ToLower(attr)
// 		}
// 		cogCfg.UserAttributes = attrs
// 	}

// 	return cogCfg, nil
// }

// type CognitoUserManager struct {
// 	Client *Cognito
// 	Config *CognitoConfig
// }

// type CognitoConfig struct {
// 	PoolID         string
// 	Region         string
// 	UserAttributes []string
// }

// func NewCognitoUserManager(cfg *CognitoConfig) (*CognitoUserManager, error) {
// 	if cfg.PoolID == "" {
// 		return nil, fmt.Errorf("invalid PoolID")
// 	}
// 	if cfg.Region == "" {
// 		return nil, fmt.Errorf("invalid Region")
// 	}
// 	if len(cfg.UserAttributes) == 0 {
// 		return nil, fmt.Errorf("no user attributes specified")
// 	}

// 	cog := NewCognito(cfg.Region, cfg.PoolID)
// 	cog.UserAttributes = cfg.UserAttributes
// 	return &CognitoUserManager{
// 		Client: cog,
// 		Config: cfg,
// 	}, nil
// }

// func (c *CognitoUserManager) ListUsers(ctx context.Context, page interface{}, filter interface{}) ([]*models.User, interface{}, error) {
// 	pageStr := page.(string)
// 	filterStr := filter.(string)

// 	users, nextPage, err := c.Client.ListUsers(filterStr, pageStr)
// 	return users, nextPage, err
// }

// // Retrieves a specific user.
// func (c *CognitoUserManager) GetUser(ctx context.Context, uid string) (*models.User, error) {
// 	return c.Client.GetUser(uid)
// }

// // NewUser creates a new user. A password is required. If a password is not specified then one will be generated.
// // TODO: FIgure out user definition.. claims.. users attributes... etc.
// func (c *CognitoUserManager) NewUser(ctx context.Context, newUser *models.User) (*models.User, error) {
// 	err := c.Client.CreateUser(newUser)
// 	return newUser, err
// }

// func (c *CognitoUserManager) UpdateUser(ctx context.Context, usr *models.User) error {
// 	return c.Client.UpdateUser(usr)
// }

// func (c *CognitoUserManager) Enable(ctx context.Context, uid string) error {
// 	return c.Client.EnableUser(uid)
// }

// func (c *CognitoUserManager) Disable(ctx context.Context, uid string) error {
// 	return c.Client.DisableUser(uid)
// }

// func (c *CognitoUserManager) DeleteUser(ctx context.Context, uid string) error {
// 	return c.Client.DeleteUser(uid)
// }

// func (c *CognitoUserManager) ForceUserName(ctx context.Context, name string) (string, bool, error) {
// 	// No validation right now

// 	u, err := c.GetUser(ctx, name)
// 	if err != nil {
// 		return name, false, err
// 	}

// 	if u != nil {
// 		return name, true, nil
// 	}

// 	return name, false, nil
// }
