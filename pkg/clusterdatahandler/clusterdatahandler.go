package clusterdatahandler

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	yaml "gopkg.in/yaml.v3"
)

type MysqlRole string

const (
	Source          MysqlRole = "source"
	Replica         MysqlRole = "replica"
	ReadonlyReplica MysqlRole = "readonly_replica"
)

const etcdClusterDataKey = "cluster_data"

type MysqlData struct {
	Role                 MysqlRole
	User, Password, Host string
	Port                 int
}

type ClusterMysqls map[string]MysqlData

type clusterDataRaw struct {
	Mysqls    map[string]MysqlData
	Proxysqls []map[string]string
	Status    map[string]string
	Users     map[string]string
}

type ClusterDataHandler struct {
	etcdClient *clientv3.Client
	etcdPrefix string
}

func NewClusterDataHandler(etcdHost, etcdUser, etcdPassword, etcdPrefix string) (*ClusterDataHandler, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdHost},
		Username:    etcdUser,
		Password:    etcdPassword,
		DialTimeout: 20 * time.Second,
		MaxUnaryRetries: 5,
		BackoffWaitBetween: 10 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating etcd client: %w", err)
	}
	return &ClusterDataHandler{etcdClient: client, etcdPrefix: etcdPrefix}, nil
}

func (cdh *ClusterDataHandler) ReadClusterMysqls(ctx context.Context) ClusterMysqls {
	cdRaw := clusterDataRaw{}
	tctx, cancel := context.WithTimeout(ctx, 5 * time.Second)
	resp, err := cdh.etcdClient.Get(tctx, cdh.etcdPrefix+etcdClusterDataKey)
	cancel()
	if err != nil {
		// TODO: check if it is better to use log
		log.Printf("%v", err)
		return cdRaw.Mysqls
	}

	if len(resp.Kvs) == 0 {
		return cdRaw.Mysqls
	}
	err = yaml.Unmarshal(resp.Kvs[0].Value, &cdRaw)
	if err != nil {
		log.Printf("There was an error unmarshaling data from etcd: %v\n", err)
	}

	return cdRaw.Mysqls
}

func (cdh *ClusterDataHandler) Destroy() {
	cdh.etcdClient.Close()
}
