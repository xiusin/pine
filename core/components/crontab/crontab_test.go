package crontab

import (
	"fmt"
	"testing"
	"time"
)

func TestCrontab_RunAll(t *testing.T) {
	cron := New()
	PrintDoc()
	cron.MustAddJob("*/2 * * * * *", func() {
		fmt.Println("hello name cur: ", time.Now().Second())
	})

	select {}
}
