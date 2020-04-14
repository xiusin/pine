package main

import (
	"fmt"
	"github.com/xiusin/pine"
	"io/ioutil"
)

func main()  {
	tmpFile, err := ioutil.TempFile("", "*")
	if err != nil {
		panic(err)
	}

	pine.RegisterOnInterrupt(func() {
		if tmpFile != nil {
			tmpFile.Close()
		}
		// DB.Close()
		// cache.Close()
		fmt.Println(" server was closed")
	})
	pine.New().Run(pine.Addr(":9528"))
}
