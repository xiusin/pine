package oss

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/xiusin/router/components/storage"
	"io"
	"strings"
)

type Option struct {
	Endpoint            string
	CustomDomain        string
	AccessKeyId         string
	AccessKeySecret     string
	BucketName          string
	PutReturnWithDomain bool
	PutOpt              []oss.Option
}

func (o *Option) GetEndpoint() string {
	return o.Endpoint
}

type Oss struct {
	client *oss.Client
	option *Option
}

func (o *Oss) PutFromReader(storeFilePath string, localPathReader io.Reader) (string, error) {
	b, err := o.client.Bucket(o.option.BucketName)
	if err != nil {
		return "", err
	}
	err = b.PutObject(storeFilePath, localPathReader, o.option.PutOpt...)
	if err != nil {
		return "", err
	}
	if o.option.PutReturnWithDomain {
		if o.option.CustomDomain == "" {
			return o.option.Endpoint + storeFilePath, nil
		} else {
			return o.option.CustomDomain + storeFilePath, nil
		}
	} else {
		return storeFilePath, nil
	}
}

func (o *Oss) PutFromFile(storeFilePath, filePath string) (string, error) {
	b, err := o.client.Bucket(o.option.BucketName)
	if err != nil {
		return "", err
	}
	s := strings.ToLower(filePath)
	// 判断是不是URL资源
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		err = b.PutObjectFromFileWithURL(storeFilePath, filePath, o.option.PutOpt...)
	} else {
		err = b.PutObjectFromFile(storeFilePath, filePath, o.option.PutOpt...)
	}
	if err != nil {
		return "", err
	}
	if o.option.PutReturnWithDomain {
		if o.option.CustomDomain == "" {
			return o.option.Endpoint + storeFilePath, nil
		} else {
			return o.option.CustomDomain + storeFilePath, nil
		}
	} else {
		return storeFilePath, nil
	}
}

func (o *Oss) Delete(storeFilePath string) error {
	b, err := o.client.Bucket(o.option.BucketName)
	if err != nil {
		return err
	}
	return b.DeleteObject(storeFilePath)
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

func (o *Oss) Bucket() (*oss.Bucket, error) {
	return o.client.Bucket(o.option.BucketName)
}

func (o *Oss) Exists(storageFilePath string) (bool, error) {
	b, err := o.Bucket()
	if err != nil {
		return false, err
	}
	return b.IsObjectExist(storageFilePath)
}

func (o *Oss) List(dir ...string) (names []string, raws oss.ListObjectsResult, err error) {
	b, err := o.Bucket()
	if err != nil {
		return
	}
	delimiter := ""
	if len(dir) == 0 {
		dir = append(dir, "")
		if dir[0] != "" {
			delimiter = "/"
		}
	}
	raws, err = b.ListObjects(oss.Delimiter(delimiter), oss.Prefix(dir[0]), oss.MaxKeys(100))
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
	storage.Register("oss", func(option storage.Option) storage.Storage {
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
