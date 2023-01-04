package cloudyaws

// import (
// 	// "crypto/x509"
// 	// "encoding/pem"
// 	// "fmt"
// 	"testing"

// 	"github.com/appliedres/cloudy"
// 	"github.com/appliedres/cloudy/testutil"
// 	cloudyvm "github.com/appliedres/cloudy/vm"
// 	"github.com/stretchr/testify/assert"
// 	// "golang.org/x/crypto/ssh"
// )

// func TestLinuxVMCreate(t *testing.T) {
// 	ctx := cloudy.StartContext()

// 	_ = testutil.LoadEnv("test.env")
// 	// vaultUrl := cloudy.ForceEnv("AZ_VAULT_URL", "")
// 	// creds := GetAzureCredentialsFromEnv(cloudy.DefaultEnvironment)

// 	// kve, _ := NewKeyVaultEnvironmentService(ctx, vaultUrl, creds, "")

// 	// env := cloudy.NewTieredEnvironment(
// 	// 	cloudy.NewTestFileEnvironmentService(),
// 	// 	kve,
// 	// )

// 	// tenantID, _ := env.Get("AZ_TENANT_ID")
// 	// ClientID, _ := env.Get("AZ_CLIENT_ID")
// 	// ClientSecret, _ := env.Get("AZ_CLIENT_SECRET")
// 	// SubscriptionID, _ := env.Get("AZ_SUBSCRIPTION_ID")
// 	// resourceGroup, _ := env.Get("AZ_RESOURCE_GROUP")
// 	// vNet, _ := env.Get("AZ_VNET")
// 	// subnet, _ := env.Get("AZ_SUBNET")
// 	// nsgName, _ := env.Get("AZ_NSG_NAME")
// 	// imageGallery, _ := env.Get("VMC_AZ_SOURCE_IMAGE_GALLERY_NAME")
// 	// nsgId, _ := env.Get("AZ_NSG_ID")

// 	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{
// 		// AzureCredentials: AzureCredentials{
// 		// 	TenantID:     tenantID,
// 		// 	ClientID:     ClientID,
// 		// 	ClientSecret: ClientSecret,
// 		// 	Region:       "usgovvirginia",
// 		// },
// 		// SubscriptionID: SubscriptionID,

// 		// ResourceGroup:            resourceGroup,
// 		// NetworkResourceGroup:     resourceGroup,
// 		// SourceImageGalleryName:   imageGallery,
// 		// Vnet:                     vNet,
// 		// AvailableSubnets:         []string{subnet},
// 		// NetworkSecurityGroupName: nsgName,
// 		// NetworkSecurityGroupID:   "NOT SET",
// 		// SaltCmd:                  "TESTSALT",
// 		// VaultURL:                 vaultUrl,
// 	})
// 	assert.Nil(t, err)

// 	// vmc.GetVMSize(ctx, "asdfaf")

// 	// cache := &AzureVMSizeCache{}
// 	// _ = cache.Load(ctx, vmc)

// 	sshPublicKey, err := env.Get("SALT_PUBLIC_KEY")
// 	assert.Nil(t, err)
// 	assert.NotNil(t, sshPublicKey)

// 	sshPrivateKey, err := env.Get("SALT_PRIVATE_KEY")
// 	assert.Nil(t, err)
// 	assert.NotNil(t, sshPrivateKey)

// 	/*
// 	   "imageReference": {
// 	   	"publisher": "canonical",
// 	   	"offer": "ubuntuserver",
// 	   	"sku": "19.04",
// 	   	"version": "19.04.202001220"
// 	   },
// 	*/
// 	vmConfig := &cloudyvm.VirtualMachineConfiguration{
// 		ID:   "uvm-gotest",
// 		Name: "uvm-gotest",
// 		Size: &cloudyvm.VmSize{
// 			Name: "Standard_DS1_v2",
// 		},
// 		SizeRequest: &cloudyvm.VmSizeRequest{
// 			SpecificSize: "Standard_DS1_v2",
// 		},
// 		OSType:       "linux",
// 		Image:        "canonical::ubuntuserver::19.04",
// 		ImageVersion: "19.04.202001220",
// 		Credientials: cloudyvm.Credientials{
// 			AdminUser:     "salt",
// 			AdminPassword: "TestPassword12#$",
// 			SSHKey:        sshPublicKey,
// 		},
// 	}

// 	// Test subnet
// 	// subnet, err := vmc.FindBestSubnet(ctx, []string{"go-on-azure-vmSubnet"})
// 	assert.Nil(t, err)
// 	assert.NotEqual(t, "", subnet)

// 	assert.NotNil(t, vmConfig.Size)

// 	// Test Create NIC
// 	err = vmc.CreateNIC(ctx, vmConfig, subnet)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, vmConfig.PrimaryNetwork)
// 	assert.NotNil(t, vmConfig.PrimaryNetwork.ID)
// 	assert.NotNil(t, vmConfig.PrimaryNetwork.Name)
// 	assert.NotNil(t, vmConfig.PrimaryNetwork.PrivateIP)

// 	defer vmc.DeleteNIC(ctx, vmConfig.ID, vmConfig.PrimaryNetwork.Name)

// 	// Test Create
// 	err = vmc.CreateVirtualMachine(ctx, vmConfig)
// 	assert.Nil(t, err)

// 	if err == nil {
// 		// block, _ := pem.Decode([]byte(sshPrivateKey))
// 		// assert.NotNil(t, block)

// 		// key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
// 		// assert.Nil(t, err)

// 		// signer, err := ssh.NewSignerFromKey(key)
// 		// assert.Nil(t, err)
// 		// assert.NotNil(t, signer)

// 		// config := &ssh.ClientConfig{
// 		// 	User:            vmConfig.Credientials.AdminUser,
// 		// 	Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
// 		// 	HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 		// }

// 		// addr := fmt.Sprintf("%s:22", vmConfig.PrimaryNetwork.PublicIP)
// 		// conn, err := ssh.Dial("tcp", addr, config)
// 		// assert.Nil(t, err)
// 		// defer conn.Close()

// 		// session, err := conn.NewSession()
// 		// assert.Nil(t, err)
// 		// session.Close()

// 		err = vmc.DeleteVM(ctx, vmConfig)
// 		assert.Nil(t, err)
// 	}

// }

// func TestWindowsVMCreate(t *testing.T) {
// 	ctx := cloudy.StartContext()
// 	_ = testutil.LoadEnv("test.env")

// 	tenantID := cloudy.ForceEnv("TenantID", "")
// 	ClientID := cloudy.ForceEnv("ClientID", "")
// 	ClientSecret := cloudy.ForceEnv("ClientSecret", "")
// 	SubscriptionID := cloudy.ForceEnv("SUBSCRIPTION_ID", "")

// 	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{
// 		// AzureCredentials: AzureCredentials{
// 		// 	TenantID:     tenantID,
// 		// 	ClientID:     ClientID,
// 		// 	ClientSecret: ClientSecret,
// 		// 	Region:       "usgovvirginia",
// 		// },
// 		// SubscriptionID: SubscriptionID,

// 		// ResourceGroup:            "go-on-azure",
// 		// NetworkResourceGroup:     "go-on-azure",
// 		// SourceImageGalleryName:   "testimagegallery",
// 		// Vnet:                     "go-on-azure-vmVNET",
// 		// AvailableSubnets:         []string{"go-on-azure-vmSubnet"},
// 		// NetworkSecurityGroupName: "go-on-azure-vmNSG",
// 		// NetworkSecurityGroupID:   "NOT SET",
// 		// SaltCmd:                  "",
// 		// VaultURL:                 "https://gokeyvault.vault.usgovcloudapi.net/",
// 	})
// 	assert.Nil(t, err)

// 	// vmc.GetVMSize(ctx, "asdfaf")

// 	// cache := &AzureVMSizeCache{}
// 	// _ = cache.Load(ctx, vmc)

// 	/*
// 	   "imageReference": {
// 	   	"publisher": "MicrosoftWindowsDesktop",
// 	   	"offer": "Windows-10",
// 	   	"sku": "21h1-ent",
// 	   	"version": "latest"
// 	   },
// 	*/
// 	vmConfig := &cloudyvm.VirtualMachineConfiguration{
// 		ID:   "uvm-gotest",
// 		Name: "uvm-gotest",
// 		Size: &cloudyvm.VmSize{
// 			Name: "Standard_DS1_v2",
// 		},
// 		SizeRequest: &cloudyvm.VmSizeRequest{
// 			SpecificSize: "Standard_DS1_v2",
// 		},
// 		OSType:       "windows",
// 		Image:        "MicrosoftWindowsDesktop::Windows-10::21h1-ent",
// 		ImageVersion: "latest",
// 		Credientials: cloudyvm.Credientials{
// 			AdminUser:     "testadmin",
// 			AdminPassword: "TestPassword12#$",
// 		},
// 	}

// 	// Test subnet
// 	subnet, err := vmc.FindBestSubnet(ctx, []string{"go-on-azure-vmSubnet"})
// 	assert.Nil(t, err)
// 	assert.Equal(t, "go-on-azure-vmSubnet", subnet)

// 	// Test Create NIC
// 	err = vmc.CreateNIC(ctx, vmConfig, subnet)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, vmConfig.PrimaryNetwork)
// 	assert.NotNil(t, vmConfig.PrimaryNetwork.ID)
// 	assert.NotNil(t, vmConfig.PrimaryNetwork.Name)
// 	assert.NotNil(t, vmConfig.PrimaryNetwork.PrivateIP)

// 	defer vmc.DeleteNIC(ctx, vmConfig.ID, vmConfig.PrimaryNetwork.Name)

// 	// Test Create
// 	err = vmc.CreateVirtualMachine(ctx, vmConfig)
// 	assert.Nil(t, err)

// 	if err == nil {
// 		err = vmc.DeleteVM(ctx, vmConfig)
// 		assert.Nil(t, err)
// 	}

// }