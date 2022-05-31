package cloudyaws

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/route53"
)

type AWSCloudFront struct {
	sess   *session.Session
	Client *cloudfront.CloudFront
}

func NewCloudFront() *AWSCloudFront {
	cf := &AWSCloudFront{}
	cf.createSession()
	return cf
}

func (awscf *AWSCloudFront) createSession() {
	if awscf.sess != nil {
		return
	}
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	awscf.sess = sess
	Client := cloudfront.New(sess)
	awscf.Client = Client
}

func (awscf *AWSCloudFront) GetDNSName(cname string) (string, error) {
	dist, err := awscf.GetDistribution(cname)
	if err != nil || dist == nil {
		return "", err
	}

	return *dist.DomainName, err
}

func (awscf *AWSCloudFront) AppendCNAME(cname string, distId string) error {
	cfgOutput, err := awscf.Client.GetDistributionConfig(&cloudfront.GetDistributionConfigInput{
		Id: aws.String(distId),
	})
	if err != nil {
		return err
	}

	cfg := cfgOutput.DistributionConfig
	for _, alias := range cfg.Aliases.Items {
		if *alias == cname {
			// It is already there
			return nil
		}
	}

	// Add the new CNAME
	cfg.Aliases.Items = append(cfg.Aliases.Items, aws.String(cname))
	cfg.Aliases.SetQuantity(int64(len(cfg.Aliases.Items)))
	cfg.Enabled = aws.Bool(true)
	e_tag := cfgOutput.ETag

	// Update
	_, err = awscf.Client.UpdateDistribution(&cloudfront.UpdateDistributionInput{
		DistributionConfig: cfg,
		Id:                 aws.String(distId),
		IfMatch:            e_tag,
	})

	return err
}

//GetDistribution looks up a distribution based on the cname or the distribution ID
func (awscf *AWSCloudFront) GetDistribution(cname string) (*cloudfront.DistributionSummary, error) {
	output, err := awscf.Client.ListDistributions(&cloudfront.ListDistributionsInput{})
	if err != nil {
		return nil, err
	}

	for _, distribution := range output.DistributionList.Items {
		if *distribution.Id == cname {
			return distribution, nil
		}
		for _, alias := range distribution.AliasICPRecordals {
			if *alias.CNAME == cname {
				return distribution, nil
			}
		}
	}

	return nil, err
}

type AWSRoute53 struct {
	sess   *session.Session
	Client *route53.Route53
}

func NewRoute53() *AWSRoute53 {
	cf := &AWSRoute53{}
	cf.createSession()
	return cf
}

func (awsroute53 *AWSRoute53) createSession() {
	if awsroute53.sess != nil {
		return
	}
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	awsroute53.sess = sess
	Client := route53.New(sess)
	awsroute53.Client = Client
}

func (awsroute53 *AWSRoute53) GetHostedZoneID(name string) (string, error) {

	output, err := awsroute53.Client.ListHostedZones(&route53.ListHostedZonesInput{})
	if err != nil {
		return "", err
	}

	// For some reason all the domain names for route53 end in a period
	namecompare := name + "."

	for _, zone := range output.HostedZones {
		if strings.EqualFold(namecompare, *zone.Name) {
			zoneid := *zone.Id
			zoneid = strings.ReplaceAll(zoneid, "/hostedzone/", "")

			return zoneid, nil
		}
	}

	return "", err
}

func (awsroute53 *AWSRoute53) UpsertARec(zoneId string, name string, DNSName string) error {
	change := &route53.Change{
		Action: aws.String("UPSERT"),
		ResourceRecordSet: &route53.ResourceRecordSet{
			Name: aws.String(name),
			Type: aws.String("A"),
			AliasTarget: &route53.AliasTarget{
				HostedZoneId:         aws.String("Z2FDTNDATAQYW2"), // Cloud front Zone
				EvaluateTargetHealth: aws.Bool(false),
				DNSName:              aws.String(DNSName),
			},
		},
	}

	_, err := awsroute53.Client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneId),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				change,
			},
		},
	})
	return err
}
