/*
Copyright 2022 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GCPSecretManagerHandler struct {
	gcpSecretsManager       *kedav1alpha1.GCPSecretManager
	gcpSecretsManagerClient *secretmanager.Client
	ctx                     context.Context
	gcpProjectID            string
}

// NewGCPSecretManagerHandler creates a HashicorpVaultHandler object
func NewGCPSecretManagerHandler(v *kedav1alpha1.GCPSecretManager, ctx context.Context) *GCPSecretManagerHandler {
	return &GCPSecretManagerHandler{
		gcpSecretsManager: v,
		ctx:               ctx,
	}
}

// Initialize the GCP Secret Manager client
func (vh *GCPSecretManagerHandler) Initialize(client client.Client, logger logr.Logger, triggerNamespace string, secretsLister corev1listers.SecretLister) error {
	var err error

	podIdentity := vh.gcpSecretsManager.PodIdentity
	if podIdentity == nil {
		podIdentity = &kedav1alpha1.AuthPodIdentity{}
	}

	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		missingErr := fmt.Errorf("clientSecret is expected when not using a pod identity provider")
		if vh.gcpSecretsManager.Credentials == nil {
			return missingErr
		}

		clientSecretName := vh.gcpSecretsManager.Credentials.ClientSecret.ValueFrom.SecretKeyRef.Name
		clientSecretKey := vh.gcpSecretsManager.Credentials.ClientSecret.ValueFrom.SecretKeyRef.Key
		clientSecret := resolveAuthSecret(vh.ctx, client, logger, clientSecretName, triggerNamespace, clientSecretKey, secretsLister)

		if clientSecret == "" {
			return missingErr
		}

		projectID := struct {
			ID string `json:"project_id"`
		}{}
		if err := json.Unmarshal([]byte(clientSecret), &projectID); err != nil {
			return fmt.Errorf("failed to unmarshal json key, %v", err)
		}
		vh.gcpProjectID = projectID.ID

		gcpCredentials, err := google.CredentialsFromJSON(vh.ctx, []byte(clientSecret), secretmanager.DefaultAuthScopes()...)
		if err != nil {
			return fmt.Errorf("failed to get credentials from json, %v", err)
		}

		vh.gcpSecretsManagerClient, err = secretmanager.NewClient(vh.ctx, option.WithCredentials(gcpCredentials))
		if err != nil {
			return fmt.Errorf("failed to create secretmanager client, %v", err)
		}

	case kedav1alpha1.PodIdentityProviderGCP:
		if vh.gcpProjectID == "" {
			return fmt.Errorf("gcp project id needs to be provided for using the gcp pod identity, %v", err)
		}

		vh.gcpProjectID = vh.gcpSecretsManager.GCPProjectID

		if vh.gcpSecretsManagerClient, err = secretmanager.NewClient(vh.ctx); err != nil {
			return fmt.Errorf("failed to create secretmanager client: %v", err)
		}
	default:
		return fmt.Errorf("key vault does not support pod identity provider - %s", podIdentity)
	}

	return nil
}

func (vh *GCPSecretManagerHandler) Read(secretID, secretVersion string) (string, error) {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/%s", vh.gcpProjectID, secretID, secretVersion),
	}

	result, err := vh.gcpSecretsManagerClient.AccessSecretVersion(vh.ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access the secret, %v", err)
	}

	crc32c := crc32.MakeTable(crc32.Castagnoli)
	checksum := int64(crc32.Checksum(result.Payload.Data, crc32c))
	if checksum != *result.Payload.DataCrc32C {
		return "", errors.New("secret payload data corruption detected")
	}

	return string(result.Payload.Data), nil
}
