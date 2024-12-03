package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	cdh "mm-haproxy/pkg/clusterdatahandler"
	hc "mm-haproxy/pkg/haproxyconfig"
)

func main() {
	etcdHost, ok := os.LookupEnv("ETCD_HOST")
	if !ok {
		fmt.Println("ETCD_HOST environment variable is not set")
		os.Exit(1)
	}
	etcdUser, ok := os.LookupEnv("ETCD_USER")
	if !ok {
		fmt.Println("ETCD_USER environment variable is not set")
		os.Exit(1)
	}
	etcdPassword, ok := os.LookupEnv("ETCD_PASSWORD")
	if !ok {
		fmt.Println("ETCD_PASSWORD environment variable is not set")
		os.Exit(1)
	}
	etcdPrefix, ok := os.LookupEnv("ETCD_PREFIX")
	if !ok {
		fmt.Println("ETCD_PREFIX environment variable is not set")
		os.Exit(1)
	}


	cdHandler := cdh.NewClusterDataHandler(etcdHost, etcdUser, etcdPassword, etcdPrefix)
	manager := hc.NewHAProxyConfigManager()
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	go manager.Run(ctx, cdHandler)
	
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	<-signalCh
	cancelFunc()
}
