package cloudyaws

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

const CognitoUserExtra_Enabled_B = "cognito-enabled"
const CognitoUserExtra_PoolID_S = "cogntio_poolid"
const CognitoUserExtra_PreferredMFA_S = "cogntio_mfa"

type Cognito struct {
	CognitoPoolID  string
	Region         string
	sess           *session.Session
	Svc            *cognitoidentityprovider.CognitoIdentityProvider
	UserAttributes []string
}

func NewCognito(region string, poolId string) *Cognito {
	c := &Cognito{
		Region:        region,
		CognitoPoolID: poolId,
	}

	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	c.sess = sess

	c.Svc = cognitoidentityprovider.New(sess)

	return c
}

// //IsAllowed determines if a request is allowed to proceed based on the GROUP that the user MUST be a part of
// func IsAllowed(group string, ctx context.Context, event events.APIGatewayProxyRequest) (bool, error) {
// 	return true, nil
// }

//GetUser calls cognito to get a list of users. The filter and page token parameters are optional
func (c *Cognito) GetUser(username string) (*models.User, error) {
	input := &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(c.CognitoPoolID),
		Username:   aws.String(username),
	}

	result, err := c.Svc.AdminGetUser(input)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:       c.findInAttributes(result.UserAttributes, "email"),
		DisplayName: c.findInAttributes(result.UserAttributes, "name"),
		UserName:    *result.Username,
		// Created:  strfmt.DateTime(*result.UserCreateDate),
		// Updated:  strfmt.DateTime(*result.UserLastModifiedDate),
		Status: *result.UserStatus,
	}

	user.Extra = make(map[string]interface{})
	user.Extra[CognitoUserExtra_Enabled_B] = cloudy.BoolFromP(result.Enabled)
	user.Extra[CognitoUserExtra_PoolID_S] = c.CognitoPoolID
	user.Extra[CognitoUserExtra_PreferredMFA_S] = cloudy.StringFromP(result.PreferredMfaSetting, "")

	return user, nil
}

func (c *Cognito) GetUserGroups(username string) ([]*models.Group, error) {
	var groups []*models.Group

	inputGroups := &cognitoidentityprovider.AdminListGroupsForUserInput{
		UserPoolId: aws.String(c.CognitoPoolID),
		Username:   aws.String(username),
		Limit:      aws.Int64(60),
	}

	results, err := c.Svc.AdminListGroupsForUser(inputGroups)
	if err != nil {
		return nil, err
	}
	for _, grp := range results.Groups {
		groups = append(groups, &models.Group{
			ID:   *grp.GroupName,
			Name: *grp.Description,
		})
	}

	return groups, nil
}

func (c *Cognito) GetUserFromEmail(email string) (*models.User, error) {
	filter := "email = " + email
	users, _, err := c.ListUsers(filter, "")

	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, nil
	}
	if len(users) > 1 {
		return users[0], fmt.Errorf("More than 1 user found with email %v... returning first", email)
	}

	return users[0], nil
}

// Overwrites all the groups that the user is in
// and sets them to the provided groups
func (c *Cognito) SetUserGroups(username string, groups []string) error {

	existingGrps, err := c.GetUserGroups(username)
	var existing []string
	for _, g := range existingGrps {
		groups = append(groups, g.ID)
	}

	// Step 2: Determine the additions and removals that need to occur
	additions := cloudy.ArrayDisjoint(groups, existing)
	removals := cloudy.ArrayDisjoint(existing, groups)
	// additions := cloudy.DifferenceStr(groups, existing)
	// removals := cloudy.DifferenceStr(existing, groups)

	for _, add := range additions {
		err = c.AddUserToGroup(add, username)
		if err != nil {
			return err
		}
	}

	for _, remove := range removals {
		err = c.RemoveUserFromGroup(remove, username)
		if err != nil {
			return err
		}
	}

	return nil
}

//ListUsers calls cognito to get a list of users. The filter and page token parameters are optional
func (c *Cognito) ListUsers(filter string, pageToken string) ([]*models.User, string, error) {

	input := &cognitoidentityprovider.ListUsersInput{
		UserPoolId: aws.String(c.CognitoPoolID),
		Limit:      aws.Int64(60),
		// AttributesToGet: []*string{aws.String("username"), aws.String("name"), aws.String("email")},
	}

	if filter != "" {
		input.Filter = aws.String(filter)
	}
	if pageToken != "" {
		input.PaginationToken = aws.String(pageToken)
	}

	result, err := c.Svc.ListUsers(input)
	if err != nil {
		return nil, "", err
	}

	var users []*models.User

	for _, user := range result.Users {
		// emailVerifiedStr := c.findInAttributes(user.Attributes, "email_verified")
		// emailVerified, err := strconv.ParseBool(emailVerifiedStr)
		// if err != nil {
		// 	emailVerified = false
		// }

		users = append(users, &models.User{
			Email:       c.findInAttributes(user.Attributes, "email"),
			DisplayName: c.findInAttributes(user.Attributes, "name"),
			UserName:    *user.Username,
			// Created:       strfmt.DateTime(*user.UserCreateDate),
			// Updated:       strfmt.DateTime(*user.UserLastModifiedDate),
			Status: *user.UserStatus,
			// EmailVerified: emailVerified,
		})
	}

	nexttoken := ""
	if result.PaginationToken != nil {
		nexttoken = *result.PaginationToken
	}

	return users, nexttoken, nil
}

//ListUsersInGroup  calls the cognitofunction
func (c *Cognito) ListAllUsersInGroup(groupName string) ([]*models.User, error) {
	var nextPageToken string
	var members []*models.User

	for {
		input := &cognitoidentityprovider.ListUsersInGroupInput{
			UserPoolId: aws.String(c.CognitoPoolID),
			GroupName:  aws.String(groupName),
			Limit:      aws.Int64(60),
		}

		if nextPageToken != "" {
			input.NextToken = aws.String(nextPageToken)
		}

		result, err := c.Svc.ListUsersInGroup(input)
		if err != nil {
			return nil, err
		}

		for _, user := range result.Users {
			members = append(members, &models.User{
				Email:       c.findInAttributes(user.Attributes, "email"),
				DisplayName: c.findInAttributes(user.Attributes, "name"),
				UserName:    *user.Username,
				// Created:  strfmt.DateTime(*user.UserCreateDate),
				// Updated:  strfmt.DateTime(*user.UserLastModifiedDate),
				Status: *user.UserStatus,
			})
		}

		if result.NextToken != nil {
			nextPageToken = *result.NextToken
		} else {
			return members, nil
		}
	}
}

//ListUsersInGroup  calls the cognitofunction
func (c *Cognito) ListUsersInGroup(groupName string, pageToken string) ([]*models.User, string, error) {

	input := &cognitoidentityprovider.ListUsersInGroupInput{
		UserPoolId: aws.String(c.CognitoPoolID),
		GroupName:  aws.String(groupName),
		Limit:      aws.Int64(60),
	}

	if pageToken != "" {
		input.NextToken = aws.String(pageToken)
	}

	result, err := c.Svc.ListUsersInGroup(input)
	if err != nil {
		return nil, "", err
	}

	var groupNames []*models.User

	for _, user := range result.Users {
		groupNames = append(groupNames, &models.User{
			Email:       c.findInAttributes(user.Attributes, "email"),
			DisplayName: c.findInAttributes(user.Attributes, "name"),
			UserName:    *user.Username,
			// Created:  strfmt.DateTime(*user.UserCreateDate),
			// Updated:  strfmt.DateTime(*user.UserLastModifiedDate),
			Status: *user.UserStatus,
		})
	}

	nextToken := ""
	if result.NextToken != nil {
		nextToken = *result.NextToken
	}

	return groupNames, nextToken, nil
}

func (c *Cognito) findInAttributes(attributes []*cognitoidentityprovider.AttributeType, field string) string {
	for _, val := range attributes {
		if *val.Name == field {
			return *val.Value
		}
	}
	return ""
}

func (c *Cognito) IsAttributeSupported(field string) bool {
	return cloudy.ArrayIncludes(c.UserAttributes, field)
}

func (c *Cognito) ToUserAttributes(user *models.User) []*cognitoidentityprovider.AttributeType {
	var attributes []*cognitoidentityprovider.AttributeType

	if user.Company != "" && c.IsAttributeSupported("company") {
		attributes = append(attributes, c.Attr("company", user.Company))
	}
	if user.Department != "" && c.IsAttributeSupported("department") {
		attributes = append(attributes, c.Attr("department", user.Department))
	}
	if user.Email != "" && c.IsAttributeSupported(Attr_email) {
		attributes = append(attributes, c.Attr(Attr_email, user.Email))
	}
	if user.FirstName != "" && c.IsAttributeSupported(Attr_givenName) {
		attributes = append(attributes, c.Attr(Attr_givenName, user.Email))
	}
	if user.LastName != "" && c.IsAttributeSupported(Attr_familyName) {
		attributes = append(attributes, c.Attr(Attr_familyName, user.Email))
	}
	if user.MobilePhone != "" && c.IsAttributeSupported(Attr_phoneNumber) {
		attributes = append(attributes, c.Attr(Attr_phoneNumber, user.MobilePhone))
	}
	if user.OfficePhone != "" && c.IsAttributeSupported("officephone") {
		attributes = append(attributes, c.Attr("officephone", user.OfficePhone))
	}
	if user.JobTitle != "" && c.IsAttributeSupported("jobtitle") {
		attributes = append(attributes, c.Attr("jobtitle", user.JobTitle))
	}
	if user.DisplayName != "" && c.IsAttributeSupported(Attr_name) {
		attributes = append(attributes, c.Attr(Attr_name, user.DisplayName))
	}

	return attributes
}

func (c *Cognito) Attr(name string, value string) *cognitoidentityprovider.AttributeType {
	return &cognitoidentityprovider.AttributeType{
		Name:  aws.String(name),
		Value: aws.String(value),
	}
}

func (c *Cognito) CreateUser(user *models.User) error {
	if user.Password == "" {
		user.Password = generatePassword(14, 2, 2, 2)
	}

	attributes := c.ToUserAttributes(user)

	input := &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId:        aws.String(c.CognitoPoolID),
		Username:          aws.String(user.ID),
		UserAttributes:    attributes,
		TemporaryPassword: aws.String(user.Password),
	}

	_, err := c.Svc.AdminCreateUser(input)

	if err != nil {
		return err
	}

	// Now email the user

	// return err
	return nil
}

func (c *Cognito) UpdateUser(user *models.User) error {
	attributes := c.ToUserAttributes(user)

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId:     aws.String(c.CognitoPoolID),
		Username:       aws.String(user.ID),
		UserAttributes: attributes,
	}

	_, err := c.Svc.AdminUpdateUserAttributes(input)

	if err != nil {
		return err
	}

	// return err
	return nil
}

func (c *Cognito) CreateGroup(group *models.Group) error {

	id := group.ID
	data, err := json.Marshal(group)
	if err != nil {
		return err
	}

	input := &cognitoidentityprovider.CreateGroupInput{
		UserPoolId:  aws.String(c.CognitoPoolID),
		GroupName:   aws.String(id),
		Description: aws.String(string(data)),
	}

	_, err = c.Svc.CreateGroup(input)
	return err
}

func (c *Cognito) DeleteGroup(grp string) error {
	_, err := c.Svc.DeleteGroup(&cognitoidentityprovider.DeleteGroupInput{
		UserPoolId: aws.String(c.CognitoPoolID),
		GroupName:  aws.String(grp),
	})

	return err
}

//ListGroups  calls the cognitofunction
func (c *Cognito) ListGroups() ([]*models.Group, error) {
	input := &cognitoidentityprovider.ListGroupsInput{
		UserPoolId: aws.String(c.CognitoPoolID),
		Limit:      aws.Int64(60),
	}

	result, err := c.Svc.ListGroups(input)
	if err != nil {
		return nil, err
	}

	var groupNames []*models.Group

	for _, group := range result.Groups {
		var grp models.Group
		data := *group.Description
		if strings.HasPrefix(data, "{") {
			err := json.Unmarshal([]byte(data), &grp)
			if err != nil {
				fmt.Printf("Error during group unmarshal: %v\n", err)
				grp = models.Group{}
			}
		} else {
			grp = models.Group{}
		}
		grp.ID = *group.GroupName
		groupNames = append(groupNames, &grp)
	}

	return groupNames, nil
}

func (c *Cognito) GetGroup(grpId string) (*models.Group, error) {
	out, err := c.Svc.GetGroup(&cognitoidentityprovider.GetGroupInput{
		UserPoolId: aws.String(c.CognitoPoolID),
		GroupName:  aws.String(grpId),
	})
	if err != nil {
		return nil, err
	}
	grp := &models.Group{}
	grp.ID = *out.Group.GroupName
	grp.Name = *out.Group.Description

	return grp, nil
}

//AddUserToGroup  calls the cognitofunction
func (c *Cognito) AddUserToGroup(groupName string, uid string) error {
	input := &cognitoidentityprovider.AdminAddUserToGroupInput{
		GroupName:  aws.String(groupName),
		UserPoolId: aws.String(c.CognitoPoolID),
		Username:   aws.String(uid),
	}

	_, err := c.Svc.AdminAddUserToGroup(input)
	return err
}

//RemoveUserFromGroup calls the cognitofunction
func (c *Cognito) RemoveUserFromGroup(groupName string, uid string) error {

	input := &cognitoidentityprovider.AdminRemoveUserFromGroupInput{
		GroupName:  aws.String(groupName),
		UserPoolId: aws.String(c.CognitoPoolID),
		Username:   aws.String(uid),
	}

	_, err := c.Svc.AdminRemoveUserFromGroup(input)
	return err
}

func (c *Cognito) ResetUserPassword(uid string) error {
	_, err := c.Svc.AdminResetUserPassword(&cognitoidentityprovider.AdminResetUserPasswordInput{
		UserPoolId: aws.String(c.CognitoPoolID), Username: aws.String(uid)})
	return err
}

func (c *Cognito) SetUserPassword(uid string, password string, permanent bool) error {
	_, err := c.Svc.AdminSetUserPassword(&cognitoidentityprovider.AdminSetUserPasswordInput{
		UserPoolId: &c.CognitoPoolID,
		Username:   &uid,
		Password:   &password,
		Permanent:  &permanent,
	})

	return err
}

func (c *Cognito) VerifyEmail(uid string) error {
	_, err := c.Svc.AdminUpdateUserAttributes(&cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: &c.CognitoPoolID,
		Username:   &uid,
		UserAttributes: []*cognitoidentityprovider.AttributeType{
			&cognitoidentityprovider.AttributeType{
				Name:  aws.String("email_verified"),
				Value: aws.String("true"),
			},
		},
	})

	return err
}

func (c *Cognito) AddCallbackUrl(domain string, userpoolid string, clientId string) error {
	describeOut, err := c.Svc.DescribeUserPoolClient(&cognitoidentityprovider.DescribeUserPoolClientInput{
		UserPoolId: aws.String(userpoolid),
		ClientId:   aws.String(clientId),
	})

	client := describeOut.UserPoolClient

	// Make sure it is not already there
	url := strings.ToLower(domain)
	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}
	if !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	for _, existingUrl := range client.CallbackURLs {
		if strings.EqualFold(*existingUrl, url) {
			return nil
		}
	}

	newCallbackURLs := append(client.CallbackURLs, aws.String(url))
	newLogoutURLs := append(client.LogoutURLs, aws.String(url))

	_, err = c.Svc.UpdateUserPoolClient(&cognitoidentityprovider.UpdateUserPoolClientInput{
		AccessTokenValidity:             client.AccessTokenValidity,
		AllowedOAuthFlows:               client.AllowedOAuthFlows,
		AllowedOAuthFlowsUserPoolClient: client.AllowedOAuthFlowsUserPoolClient,
		AllowedOAuthScopes:              client.AllowedOAuthScopes,
		AnalyticsConfiguration:          client.AnalyticsConfiguration,
		ClientId:                        client.ClientId,
		ClientName:                      client.ClientName,
		DefaultRedirectURI:              client.DefaultRedirectURI,
		ExplicitAuthFlows:               client.ExplicitAuthFlows,
		IdTokenValidity:                 client.IdTokenValidity,
		PreventUserExistenceErrors:      client.PreventUserExistenceErrors,
		ReadAttributes:                  client.ReadAttributes,
		RefreshTokenValidity:            client.RefreshTokenValidity,
		SupportedIdentityProviders:      client.SupportedIdentityProviders,
		TokenValidityUnits:              client.TokenValidityUnits,
		UserPoolId:                      client.UserPoolId,
		WriteAttributes:                 client.WriteAttributes,
		CallbackURLs:                    newCallbackURLs,
		LogoutURLs:                      newLogoutURLs,
	})

	return err
}

func (c *Cognito) EnableUser(uid string) error {
	_, err := c.Svc.AdminEnableUser(&cognitoidentityprovider.AdminEnableUserInput{
		UserPoolId: aws.String(c.CognitoPoolID),
		Username:   aws.String(uid),
	})

	return err
}

func (c *Cognito) DisableUser(uid string) error {
	_, err := c.Svc.AdminDisableUser(&cognitoidentityprovider.AdminDisableUserInput{
		UserPoolId: aws.String(c.CognitoPoolID),
		Username:   aws.String(uid),
	})

	return err
}

func (c *Cognito) DeleteUser(uid string) error {
	_, err := c.Svc.AdminDeleteUser(&cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: aws.String(c.CognitoPoolID),
		Username:   aws.String(uid),
	})

	return err
}

// func (c *Cognito) ProcessUserActions(actions []*models.UpdateUserAction) error {

// 	for _, action := range actions {
// 		switch action.Action {
// 		case "verify_email":
// 			err := c.VerifyEmail(action.Username)
// 			if err != nil {
// 				return err
// 			}
// 		case "reset_password":
// 			err := c.ResetUserPassword(action.Username)
// 			if err != nil {
// 				return err
// 			}
// 		case "set_password":
// 			permenant, _ := strconv.ParseBool(action.ActionArg2)
// 			err := c.SetUserPassword(action.Username, action.ActionArg1, permenant)
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	return nil
// }

var (
	lowerCharSet   = "abcdedfghijklmnopqrst"
	upperCharSet   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	specialCharSet = "!@#$%&*"
	numberSet      = "0123456789"
	allCharSet     = lowerCharSet + upperCharSet + specialCharSet + numberSet
)

func generatePassword(passwordLength, minSpecialChar, minNum, minUpperCase int) string {
	var password strings.Builder

	//Set special character
	for i := 0; i < minSpecialChar; i++ {
		random := rand.Intn(len(specialCharSet))
		password.WriteString(string(specialCharSet[random]))
	}

	//Set numeric
	for i := 0; i < minNum; i++ {
		random := rand.Intn(len(numberSet))
		password.WriteString(string(numberSet[random]))
	}

	//Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random := rand.Intn(len(upperCharSet))
		password.WriteString(string(upperCharSet[random]))
	}

	remainingLength := passwordLength - minSpecialChar - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random := rand.Intn(len(allCharSet))
		password.WriteString(string(allCharSet[random]))
	}
	inRune := []rune(password.String())
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}
