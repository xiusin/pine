package oss

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/xiusin/router/components/storage"
	"sync"
)

type Option struct {
	Endpoint        string
	CustomDomain    string
	AccessKeyId     string
	AccessKeySecret string
	BucketName      string
	PutOpt          []oss.Option
}

func (o *Option) GetEndpoint() string {
	return o.Endpoint
}

type Oss struct {
	client *oss.Client
	option *Option
	mu     sync.Mutex
}

func (o *Oss) Put(storeFilePath, localPath string) (string, error) {
	b, err := o.client.Bucket(o.option.BucketName)
	if err != nil {
		return "", err
	}
	err = b.PutObjectFromFile(storeFilePath, localPath, o.option.PutOpt...)
	if err != nil {
		return "", err
	}
	if o.option.CustomDomain == "" {
		return o.option.Endpoint + storeFilePath, nil
	} else {
		return o.option.CustomDomain + storeFilePath, nil
	}
}

func (o *Oss) Delete(storeFilePath string) error {
	b, err := o.client.Bucket(o.option.BucketName)
	if err != nil {
		return err
	}
	return b.DeleteObject(storeFilePath, o.option.PutOpt...)
}

func (o *Oss) ListBucket() (names []string, raws oss.ListBucketsResult, err error) {
	raws, err = o.client.ListBuckets()
	if err != nil {
		return
	}
	for k, _ := range raws.Buckets {
		names = append(names, raws.Buckets[k].Name)
	}
	return
}

func (o *Oss) Client() *oss.Client {
	return o.client
}

func (o *Oss) List(dir ...string) (names []string, raws oss.ListObjectsResult, err error) {
	b, err := o.client.Bucket(o.option.BucketName)
	if err != nil {
		return
	}
	raws, err = b.ListObjects(o.option.PutOpt...)
	if err != nil {
		return
	}
	for k, _ := range raws.Objects {
		names = append(names, raws.Objects[k].Key)
	}
	return
}

func (o *Option) checkValid() bool {
	if o.BucketName == "" || o.AccessKeySecret == "" || o.AccessKeyId == "" || o.Endpoint == "" {
		return false
	} else {
		return true
	}
}

func init() {
	storage.Register("oss", func(option storage.Option) storage.StorageInf {
		opt := option.(*Option)
		if !opt.checkValid() {
			panic("oss option is not valid")
		}
		client, err := oss.New(opt.GetEndpoint(), opt.AccessKeyId, opt.AccessKeySecret)
		if err != nil {
			panic(err)
		}
		ok, err := client.IsBucketExist(opt.BucketName)
		if err != nil {
			panic(err)
		}
		if !ok {
			err = client.CreateBucket(opt.BucketName)
		}
		if err != nil {
			panic(err)
		}
		instance := &Oss{client: client, option: opt}
		return instance
	})
}
