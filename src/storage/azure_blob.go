package storage

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

var containerClient *azblob.ContainerClient

func InitAzure(account, key string) error {
	cred, err := azblob.NewSharedKeyCredential(account, key)
	if err != nil {
		return err
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", account)

	serviceClient, err := azblob.NewServiceClientWithSharedKey(serviceURL, cred, nil)
	if err != nil {
		return err
	}

	containerClient = serviceClient.NewContainerClient("icpc-testcases")
	return nil
}

func UploadFile(path string, data []byte) (string, error) {
	blob := containerClient.NewBlockBlobClient(path)

	_, err := blob.UploadBuffer(context.TODO(), data, nil)
	if err != nil {
		return "", err
	}

	return blob.URL(), nil
}