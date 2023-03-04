package registry

require (
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.2.1"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.0.0"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type BlobRegistry struct {
	account string
	container string
}

func (r *BlobRegistry) ListModules(namespace, name, provider string, offset, limit int) (modules []models.Module, total int, err error) {
	modules, err = r.getModules(namespace, name, provider)

	if err != nil {
		return nil, 0, err
	}

	return modules, len(modules), nil
}

func (BlobRegistry) PublishModule(namepsace, name, provider, version string, data io.Reader) (err error) {
	panic("implement me")
}

func (r *BlobRegistry) GetModuleData(namespace, name, provider, version string) (reader *bytes.Buffer, err error) {

	client, err := GetClient(r)
	handleError(err)

	obj, err := client.DownloadStream(
		ctx, 
		r.container, 
		strings.Join([]string{namespace, name, provider, version}, "/") + ".tgz", 
		nil)

	handleError(err)

	if err != nil {
		return nil, err
	}

	buffer := &bytes.Buffer{}
	io.Copy(buffer, obj.Body)
	return buffer, nil
}

func (r *BlobRegistry) getModules(namespace, name, provider string) (modules []models.Module, err error) {

	client, err := GetClient(r)
	handleError(err)

	prefix := ""

	if namespace != "" {
		prefix = namespace
		if name != "" {
			prefix = strings.Join([]string{prefix, name}, "/")
			if provider != "" {
				prefix = strings.Join([]string{prefix, provider}, "/")
			}
		}
	}

	if prefix != "" {
		prefix += "/"
	}

	//fmt.Println("Listing the blobs in the container:")

	pager := client.NewListBlobsFlatPager(r.container, &azblob.ListBlobsFlatOptions{
		Include: azblob.ListBlobsInclude{Snapshots: true, Versions: true},
		Prefix: prefix
	})

	for pager.More() {
		resp, err := pager.NextPage(context.TODO())
		handleError(err)

		for _, blob := range resp.Segment.BlobItems {
			parts := strings.Split(*blob.Name, "/")

			if len(parts) == 4 {
				modules = append(modules, models.Module{
					Namespace: parts[0],
					Name:      parts[1],
					Provider:  parts[2],
					Version:   strings.TrimRight(parts[3], ".tgz"),
				})
			}
		}
	}

	return modules, nil
}

func (r *BlobRegistry) getClient() *Client {

	url := fmt.Printf("https://%v.blob.core.windows.net/", r.account)
	ctx := context.Background()

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	handleError(err)

	client, err := azblob.NewClient(url, credential, nil)
	handleError(err)

	return client
}

func NewBlobRegistry(options app.BlobOptions) Registry {
	return &BlobRegistry{
		options.Account,
		options.Container
	}
}
