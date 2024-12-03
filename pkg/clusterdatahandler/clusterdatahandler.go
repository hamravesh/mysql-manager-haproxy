package clusterdatahandler

import (
	"context"
)

type MysqlRole string 

const (
	Source MysqlRole = "source" 
	Replica MysqlRole = "replica" 
	ReadonlyReplica MysqlRole = "readonly_replica" 
)


type MysqlData struct {
	role MysqlRole
	user, password, host string 
	port int
}

type ClusterData struct {
	Mysqls map[string]MysqlData
}

type ClusterDataHandler struct {
	EtcdHost     string
	EtcdUser     string
	EtcdPassword string
	EtcdPrefix   string
}

func NewClusterDataHandler(host, user, password, prefix string) ClusterDataHandler {
	return ClusterDataHandler{
		EtcdHost: host,
		EtcdUser: user,
		EtcdPassword: password,
		EtcdPrefix: prefix,
	}
}

func (cdh *ClusterDataHandler) ReadClusterData(ctx context.Context) ClusterData {
	return ClusterData{}
}
