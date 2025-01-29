package haproxyconfig

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"text/template"

	cdh "mm-haproxy/pkg/clusterdatahandler"
	"time"
)

// TODO: support readonly mysql servers
const tmplText = `global
    maxconn 10000
	master-worker no-exit-on-failure
    stats socket /var/lib/haproxy/haproxy.sock expose-fd listeners level admin

defaults
    log global
    mode tcp
    retries 10
    default-server init-addr last,libc,none
	timeout client 1000s
    timeout connect 1000s
    timeout server 1000s

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
    option srvtcpka
{{- if .ReplicaHost }}
    server repl {{.ReplicaHost}}:3306 check
{{- end }}

backend mysql-source
    mode tcp
    option srvtcpka
{{- if .SourceHost }}
    server src {{.SourceHost}}:3306 check
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
	log.Println("Started haproxy process")

	time.Sleep(5*time.Second)
	
	ticker := time.NewTicker(time.Duration(hcm.ClusterDataCheckInterval) * time.Second)
	var haproxyNeedsReload bool
	for {
		select {
		case <-signalCh:
			log.Println("Shutting down...")
			cancelFunc()
			cmd.Process.Kill()
		case <-ticker.C:
			log.Println("Reading data from etcd")
			haproxyNeedsReload = false
			newMysqls := cdHandler.ReadClusterMysqls(ctx)
			for _, m := range newMysqls {
				if m.Role == cdh.Source {
					log.Printf("Current source is: %s", m.Host)
					if m.Host != haproxyConfigData.SourceHost {
						log.Printf("Changing source to: %s", m.Host)
						haproxyConfigData.SourceHost = m.Host
						haproxyNeedsReload = true
					}
				} else if m.Role == cdh.Replica {
					log.Printf("Current replica is: %s", m.Host)
					if m.Host != haproxyConfigData.ReplicaHost {
						log.Printf("Changing replica to: %s", m.Host)
						haproxyConfigData.ReplicaHost = m.Host
						haproxyNeedsReload = true
					}
				}
			}

			if haproxyNeedsReload {
				err := hcm.writeHAProxyConfig(haproxyConfigData)
				if err != nil {
					log.Printf("%v", err)
					continue
				}
				cmd.Process.Signal(syscall.SIGUSR2)
				time.Sleep(5*time.Second)
				log.Println("Restarted haproxy process")
			}

			log.Println("Enabling servers if needed")
			if hcm.needsReload("mysql-source") || hcm.needsReload("mysql-replica") {
				cmd.Process.Signal(syscall.SIGUSR2)
				log.Println("Reloaded haproxy process")
				time.Sleep(2*time.Second)
			}
		}
	}
}

func (hcm *HAProxyConfigManager) needsReload(backend string) bool {
	showServersCommand := fmt.Sprintf("echo 'show servers state %s' | socat stdio /var/lib/haproxy/haproxy.sock", backend)
	statusCmd := exec.Command("bash", "-c", showServersCommand)
	out, err := statusCmd.CombinedOutput()
	if err != nil {
		log.Printf("Failure in getting server state src: %v\n", err.Error())
		return false
	}
	outStr := string(out)
	lines := strings.Split(outStr, "\n")
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "1" {
			continue
		}
		fields := strings.Split(line, " ")
		log.Println(fields)
		if fields[6] == "32" {
			return true
		}
	}
	return false
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
