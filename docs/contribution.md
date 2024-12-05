## Tests
To run tests first run etcd and configure it:
```sh
docker compose up -d etcd
./tests/setup-etcd.sh
```
and run other containers: 
```sh
docker compose up -d mm
docker compose exec mm python cli/mysql-cli.py init -f /etc/mm/cluster-spec.yaml
docker compose up -d mm-hap --build
```

for restarting `mm-hap`:
```sh
docker compose down mm-hap && docker compose up -d --build mm-hap 
docker compose logs mm-hap
```