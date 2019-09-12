package option

import "time"

type IOption interface {
	ToViper()
	SetDebug()
	IsDevMode() bool
	IsProdMode() bool
	GetMaxMultipartMemory() int64
	GetHost() string
	GetGzip() bool
	GetReqTimeOutMessage() string
	GetTimeOut() time.Duration
}
