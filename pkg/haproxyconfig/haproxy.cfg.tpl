global
    maxconn 10000
    stats socket /etc/haproxy/mysql/haproxy.sock mode 600 expose-fd listeners level admin

defaults
    log global
    mode tcp
    retries 10
    timeout client 28800s
    timeout connect 100500
    timeout server 28800s

frontend mysql-source-fe
    bind *:3306
    option clitcpka
    use_backend mysql-source

frontend stats
      bind *:8484
      mode http
      http-request use-service prometheus-exporter if { path /metrics }

backend mysql-source
    mode tcp
    option srvtcpka
    server mysql-source {{.Host}}:3306 check
