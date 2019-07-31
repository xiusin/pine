package oss

import (
	"github.com/xiusin/router/components/storage"
	"testing"
)

var adapter storage.StorageInf

func init() {
	adapterTmp, err := storage.NewStorage("oss", &Option{
		Endpoint:        "oss-cn-shanghai.aliyuncs.com",
		AccessKeyId:     "LTAIdVHSv27uLaXu",
		AccessKeySecret: "sHvW947BVraoRiUCAhy0r8h0Ixd7QJ",
		BucketName:      "bucket-blog",
	})
	if err != nil {
		panic(err)
	}
	adapter = adapterTmp
}

func TestOss_ListBucket(t *testing.T) {
	oss := adapter.(*Oss)
	t.Log(oss.ListBucket())
	ls, _, _ := oss.List("bucket-blog")
	t.Log()

}
