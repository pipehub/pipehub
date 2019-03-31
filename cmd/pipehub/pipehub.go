package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
)

func fatal(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}

func wait() {
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}

func asyncErrHandler(err error) {
	fmt.Println(errors.Wrap(err, "async error occurred").Error())
	done <- syscall.SIGTERM
}
