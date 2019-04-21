package main

import (
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/xiusin/router/core"
	"io"
)

func main() {
	handler := core.NewRouter(nil)
	handler.GET("/hello/:name", func(c *core.Context) {
		conn, _, _, err := ws.UpgradeHTTP(c.Request(), c.Writer())
		if err != nil {
			// handle error
		}
		go func() {
			defer conn.Close()

			var (
				state  = ws.StateServerSide
				reader = wsutil.NewReader(conn, state)
				writer = wsutil.NewWriter(conn, state, ws.OpText)
			)
			for {
				header, err := reader.NextFrame()
				if err != nil {
					// handle error
				}

				// Reset writer to write frame with right operation code.
				writer.Reset(conn, state, header.OpCode)

				if _, err = io.Copy(writer, reader); err != nil {
					// handle error
				}
				if err = writer.Flush(); err != nil {
					// handle error
				}
			}
		}()
	})
	handler.Serve()
}
