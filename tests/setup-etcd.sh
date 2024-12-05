#!/bin/bash

docker compose exec etcd etcdctl user add root --new-user-password="password"
docker compose exec etcd etcdctl user grant-role root root
docker compose exec etcd etcdctl user add mm --new-user-password="password"
docker compose exec etcd etcdctl role add mm 
docker compose exec etcd etcdctl role grant-permission mm \
    --prefix=true readwrite mm/cluster1/
docker compose exec etcd etcdctl user grant-role mm mm
docker compose exec etcd etcdctl auth enable
