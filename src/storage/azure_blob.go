package storage

// import (
// 	"context"
// 	"fmt"

// 	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
// )

// var blobClient *azblob.Client
// var activeContainer string 


// func InitAzure(account, key, containerName string) error {
// 	cred, err := azblob.NewSharedKeyCredential(account, key)
// 	if err != nil {
// 		return err
// 	}

// 	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", account)

// 	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
// 	if err != nil {
// 		return err
// 	}

// 	blobClient = client
// 	activeContainer = containerName 
// 	return nil
// }

// func UploadFile(path string, data []byte) (string, error) {
	
// 	_, err := blobClient.UploadBuffer(context.TODO(), activeContainer, path, data, nil)
// 	if err != nil {
// 		return "", err
// 	}

// 	blobURL := fmt.Sprintf("%s%s/%s", blobClient.URL(), activeContainer, path)
// 	return blobURL, nil
// }