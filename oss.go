// GNU GPL v3 License
// Copyright (c) 2019 github.com:go-trellis

package alioss

import (
	"fmt"
	"io"
	"path"
	"strings"
	"sync"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/go-trellis/config"
	uuid "github.com/satori/go.uuid"
)

// 基本参数
const (
	AliossPrefix = "oss://"
)

// Client 客户端API说明
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

////// Default Client

type client struct {
	Trellis struct {
		AliOss struct {
			Domain        string `yaml:"domain" json:"domain"`
			EndPoint      string `yaml:"end_point" json:"end_point"`
			AccessID      string `yaml:"access_id" json:"access_id"`
			AccessKey     string `yaml:"access_key" json:"access_key"`
			ExpireSeconds int64  `yaml:"expire_seconds" json:"expire_seconds"`
		} `yaml:"alioss" json:"alioss"`
	} `yaml:"trellis" json:"trellis"`

	ossMutex   *sync.RWMutex
	ossClient  *oss.Client
	ossBuckets map[string]*oss.Bucket
}

func (p *client) init() (err error) {
	p.ossMutex = &sync.RWMutex{}
	p.ossBuckets = make(map[string]*oss.Bucket)
	p.ossClient, err = oss.New(
		p.Trellis.AliOss.EndPoint, p.Trellis.AliOss.AccessID, p.Trellis.AliOss.AccessKey)
	return
}

func (p *client) getOssBuckets(name string) (*oss.Bucket, bool) {
	p.ossMutex.RLock()
	b, ok := p.ossBuckets[name]
	p.ossMutex.RUnlock()
	return b, ok
}

func (p *client) setOssBuckets(name string, bucket *oss.Bucket) {
	p.ossMutex.Lock()
	p.ossBuckets[name] = bucket
	p.ossMutex.Unlock()
}

// NewClientFromFile 从配置文件读取
func NewClientFromFile(filePath string) (Client, error) {
	r, err := config.NewSuffixReader(config.ReaderOptionFilename(filePath))
	if err != nil {
		return nil, err
	}

	c := &client{}
	err = r.Read(c)
	if err != nil {
		return nil, err
	}

	if err := c.init(); err != nil {
		return nil, err
	}
	return c, nil
}

// NewClient 获取新的客户端
func NewClient(accessID, accessKey, endPoint string, expireSeconds int64) (Client, error) {
	c := &client{}
	c.Trellis.AliOss.AccessID = accessID
	c.Trellis.AliOss.AccessKey = accessKey
	c.Trellis.AliOss.EndPoint = endPoint
	c.Trellis.AliOss.ExpireSeconds = expireSeconds
	if err := c.init(); err != nil {
		return nil, err
	}
	return c, nil
}

func (p *client) GenObjectID(prefixPath, fileSuffix string) string {

	path := AliossPrefix + path.Join(prefixPath,
		strings.Replace(uuid.NewV4().String(), "-", "", -1))
	if len(fileSuffix) == 0 {
		return path
	}

	if fileSuffix[0:1] != "." {
		path += "."
	}
	return path + fileSuffix
}

func (p *client) PutObject(bucketName, objectID string, reader io.Reader) (err error) {
	b, ok := p.getOssBuckets(bucketName)
	if !ok {
		b, err = p.ossClient.Bucket(bucketName)
		if err != nil {
			return
		}
		p.setOssBuckets(bucketName, b)
	}
	objectID = strings.TrimPrefix(objectID, AliossPrefix)
	return b.PutObject(objectID, reader)
}

func (p *client) GetSignURL(bucketName, objectID string, options ...oss.Option) (string, error) {
	if !strings.HasPrefix(objectID, AliossPrefix) {
		return objectID, nil
	}
	objectID = strings.TrimPrefix(objectID, AliossPrefix)

	b, ok := p.getOssBuckets(bucketName)
	if !ok {
		var err error
		b, err = p.ossClient.Bucket(bucketName)
		if err != nil {
			return "", err
		}
		p.setOssBuckets(bucketName, b)
	}

	url, err := b.SignURL(objectID, oss.HTTPGet, p.Trellis.AliOss.ExpireSeconds, options...)
	if err != nil {
		return "", err
	}
	if len(p.Trellis.AliOss.Domain) != 0 {
		url = strings.Replace(url,
			fmt.Sprintf("%s.%s", bucketName, p.Trellis.AliOss.EndPoint),
			p.Trellis.AliOss.Domain, -1)
	}
	return url, nil
}

func (p *client) ListObjects(bucketName string) (res *oss.ListObjectsResult, err error) {
	b, ok := p.getOssBuckets(bucketName)
	if !ok {
		b, err = p.ossClient.Bucket(bucketName)
		if err != nil {
			return
		}
		p.setOssBuckets(bucketName, b)
	}

	result, e := b.ListObjects()
	if e != nil {
		return nil, e
	}
	return &result, nil
}

func (p *client) DeleteObject(bucketName, objectID string) (err error) {

	objectID = strings.TrimPrefix(objectID, AliossPrefix)

	b, ok := p.getOssBuckets(bucketName)
	if !ok {
		b, err = p.ossClient.Bucket(bucketName)
		if err != nil {
			return
		}
		p.setOssBuckets(bucketName, b)
	}

	return b.DeleteObject(objectID)
}
