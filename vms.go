package cloudyaws

import (
	"context"

	cloudyvm "github.com/appliedres/cloudy/vm"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicequotas"
)

type EC2Controller struct {
	Quotas *servicequotas.ServiceQuotas
}

func NewEC2Controller(ctx context.Context) *EC2Controller {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	quotas := servicequotas.New(sess)

	return &EC2Controller{
		Quotas: quotas,
	}
}

func (ec2 *EC2Controller) GetLimits(ctx context.Context) ([]*cloudyvm.VirtualMachineLimit, error) {
	// TODO: Look up current usage
	// TOOD: Match quota name or code to ec2 size

	var rtn []*cloudyvm.VirtualMachineLimit

	out, err := ec2.Quotas.ListServiceQuotas(&servicequotas.ListServiceQuotasInput{
		ServiceCode: aws.String("ec2"),
	})
	if err != nil {
		return nil, err
	}

	for {
		for _, q := range out.Quotas {
			rtn = append(rtn, &cloudyvm.VirtualMachineLimit{
				Name:  *q.QuotaName,
				Limit: int(*q.Value),
			})
		}

		if out.NextToken != nil {
			out, err = ec2.Quotas.ListServiceQuotas(&servicequotas.ListServiceQuotasInput{
				ServiceCode: aws.String("ec2"),
				NextToken:   out.NextToken,
			})

			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	return rtn, nil
}
