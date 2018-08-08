# ali_oss
api for https://github.com/aliyun/aliyun-oss-go-sdk


## Client 客户端API说明

```golang
type Client interface {
	// 生成文件Oss路径
	GenObjectID(prefixPath, filename string) string
	// 上传文件
	PutObject(bucketName, objectID string, reader io.Reader) error
	// 获取文件地址
	GetSignURL(bucketName, objectID string, options ...oss.Option) (string, error)
	// 删除文件地址
	DeleteObject(bucketName, objectID string) error
	// 获取所有object列表
	ListObjects(bucketName string) (*oss.ListObjectsResult, error)
}

client, _ := NewClientFromFile("oss.yaml")

```