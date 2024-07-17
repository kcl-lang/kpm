package downloader

import (
	"fmt"

	dockerauth "oras.land/oras-go/pkg/auth/docker"
	remoteauth "oras.land/oras-go/v2/registry/remote/auth"
)

// CredClient is the client to get the credentials.
type CredClient struct {
	credsClient *dockerauth.Client
}

// LoadCredentialFile loads the credential file and return the CredClient.
func LoadCredentialFile(filepath string) (*CredClient, error) {
	authClient, err := dockerauth.NewClientWithDockerFallback(filepath)
	if err != nil {
		return nil, err
	}
	dockerAuthClient, ok := authClient.(*dockerauth.Client)
	if !ok {
		return nil, fmt.Errorf("authClient is not *docker.Client type")
	}

	return &CredClient{
		credsClient: dockerAuthClient,
	}, nil
}

// GetAuthClient returns the auth client.
func (cred *CredClient) GetAuthClient() *dockerauth.Client {
	return cred.credsClient
}

// Credential will reture the credential info cache in CredClient
func (cred *CredClient) Credential(hostName string) (*remoteauth.Credential, error) {
	if len(hostName) == 0 {
		return nil, fmt.Errorf("hostName is empty")
	}
	username, password, err := cred.credsClient.Credential(hostName)
	if err != nil {
		return nil, err
	}

	return &remoteauth.Credential{
		Username: username,
		Password: password,
	}, nil
}
