package main

import (
	"fmt"
	"os"
	"strconv"

	cdh "mm-haproxy/pkg/clusterdatahandler"
	hc "mm-haproxy/pkg/haproxyconfig"
)

const clusterDataCheckIntervalDefault = 2

func main() {
	etcdHost, ok := os.LookupEnv("ETCD_HOST")
	if !ok {
		fmt.Println("ETCD_HOST environment variable is not set")
		os.Exit(1)
	}
	etcdUser, ok := os.LookupEnv("ETCD_USERNAME")
	if !ok {
		fmt.Println("ETCD_USERNAME environment variable is not set")
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

	cdHandler, err := cdh.NewClusterDataHandler(etcdHost, etcdUser, etcdPassword, etcdPrefix)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	clusterDataCheckInterval := clusterDataCheckIntervalDefault
	clusterDataCheckIntervalStr, ok := os.LookupEnv("CLUSTER_DATA_CHECK_INTERVAL")
	if !ok {
		clusterDataCheckInterval, err = strconv.Atoi(clusterDataCheckIntervalStr)
		if err != nil {
			clusterDataCheckInterval = clusterDataCheckIntervalDefault
		}
	}

	manager := hc.NewHAProxyConfigManager(clusterDataCheckInterval)
	manager.Run(cdHandler)
}
