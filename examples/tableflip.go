package main

import (
	"context"
	"github.com/cloudflare/tableflip"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main()  {
	upg, err := tableflip.New(tableflip.Options{})
	if err != nil {
		panic(err)
	}
	defer upg.Stop()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGHUP)
		for range sig {
			err := upg.Upgrade()
			if err != nil {
				log.Println("Upgrade failed:", err)
				continue
			}

			log.Println("Upgrade succeeded")
		}
	}()

	ln, err := upg.Fds.Listen("tcp", "127.0.0.1:9528")
	if err != nil {
		log.Fatalln("Can't listen:", err)
	}

	var server http.Server
	go server.Serve(ln)

	if err := upg.Ready(); err != nil {
		panic(err)
	}
	<-upg.Exit()

	time.AfterFunc(30*time.Second, func() {
		os.Exit(1)
	})

	_ = server.Shutdown(context.Background())
}