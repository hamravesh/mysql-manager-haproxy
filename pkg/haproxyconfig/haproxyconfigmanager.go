package haproxyconfig

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"text/template"

	cdh "mm-haproxy/pkg/clusterdatahandler"
	"time"
)

// TODO: support readonly mysql servers
const tmplText = `global
    maxconn 10000
    stats socket ipv4@127.0.0.1:9999 expose-fd listeners level admin

defaults
    log global
    mode tcp
    retries 10
    default-server init-addr last,libc,none
    timeout client 28800s
    timeout connect 100500
    timeout server 28800s

resolvers mydns
    parse-resolv-conf
    hold refused  1d
    hold nx       10d

frontend mysql-replica-fe
    bind *:3307
    option clitcpka
    use_backend mysql-replica

frontend mysql-source-fe
    bind *:3306
    option clitcpka
    use_backend mysql-source

frontend stats
    bind *:6070
    mode http
    http-request use-service prometheus-exporter if { path /metrics }

backend mysql-replica
    mode tcp
{{- if .ReplicaHost }}
    server repl {{.ReplicaHost}}:3306 check resolvers mydns
{{- end }}

backend mysql-source
    mode tcp
{{- if .SourceHost }}
    server src {{.SourceHost}}:3306 check resolvers mydns
{{- end }}
`

// TODO: write an interface for cdhandler
type HAProxyConfigManager struct {
	ClusterDataCheckInterval int
}

type HAProxyConfigData struct {
	SourceHost  string
	ReplicaHost string
}

func NewHAProxyConfigManager(interval int) HAProxyConfigManager {
	return HAProxyConfigManager{ClusterDataCheckInterval: interval}
}

func (hcm *HAProxyConfigManager) Run(cdHandler *cdh.ClusterDataHandler) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	haproxyConfigData := HAProxyConfigData{}
	hcm.writeHAProxyConfig(haproxyConfigData)

	cmd := exec.Command("haproxy", "-sf", "-f", "/var/lib/haproxy/haproxy.cfg")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	defer cmd.Wait()
	log.Println("Started haproxy process")

	ticker := time.NewTicker(time.Duration(hcm.ClusterDataCheckInterval) * time.Second)
	var haproxyNeedsRestart bool 
	for {
		select {
		case <-signalCh:
			log.Println("Shutting down...")
			cancelFunc()
			cmd.Process.Kill()
		case <-ticker.C:
			log.Println("Reading data from etcd")
			haproxyNeedsRestart = false
			newMysqls := cdHandler.ReadClusterMysqls(ctx)
			for _, m := range newMysqls {
				if m.Role == cdh.Source {
					log.Printf("Current source is: %s", m.Host)
					if m.Host != haproxyConfigData.SourceHost {
						log.Printf("Changing source to: %s", m.Host)
						haproxyConfigData.SourceHost = m.Host
						haproxyNeedsRestart = true
					}
				} else if m.Role == cdh.Replica {
					log.Printf("Current replica is: %s", m.Host)
					if m.Host != haproxyConfigData.ReplicaHost {
						log.Printf("Changing replica to: %s", m.Host)
						haproxyConfigData.ReplicaHost = m.Host
						haproxyNeedsRestart = true
					}
				}
			}

			if haproxyNeedsRestart {
				err := hcm.writeHAProxyConfig(haproxyConfigData)
				if err != nil {
					log.Printf("%v", err)
					continue
				}
				cmd.Process.Kill()
				cmd.Wait()
				cmd = exec.Command("haproxy", "-sf", "-f", "/var/lib/haproxy/haproxy.cfg")
				cmd.Stdout = os.Stdout
                cmd.Stderr = os.Stderr
				cmd.Start()
				log.Println("Restarted haproxy process")
			}
		}
	}
}

func (hcm *HAProxyConfigManager) writeHAProxyConfig(data HAProxyConfigData) error {
	tmpl, err := template.New("config").Parse(tmplText)
	if err != nil {
		return fmt.Errorf("error in creating template for config %v", err)
	}

	var f *os.File
	f, err = os.Create("/var/lib/haproxy/haproxy.cfg")
	if err != nil {
		return fmt.Errorf("could not open haproxy config file %v", err)
	}
	defer f.Close()
	err = tmpl.Execute(f, data)
	if err != nil {
		return fmt.Errorf("could not write to config file %v", err)
	}
	log.Println("HAProxy config is written to /var/lib/haproxy/haproxy.cfg")
	return nil
}
