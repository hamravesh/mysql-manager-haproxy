package haproxyconfig

import (
	"context"
	cdh "mm-haproxy/pkg/clusterdatahandler"
)

// TODO: write an interface for cdhandler
type HAProxyConfigManager struct{}

func NewHAProxyConfigManager() HAProxyConfigManager {
	return HAProxyConfigManager{}
}

func (hcm *HAProxyConfigManager) Run(ctx context.Context, cdHandler cdh.ClusterDataHandler) {
	cdHandler.ReadClusterData(ctx)
}
