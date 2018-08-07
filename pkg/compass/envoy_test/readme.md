docker build -t envoy_test .
docker run --rm -p 28080:8080 jmalloc/echo-server
docker run --rm -v $(pwd)/envoy1.yaml:/etc/envoy/envoy.yaml -p 9001:9000 -p 10001:10000 envoy_test
docker run --rm -v $(pwd)/envoy2.yaml:/etc/envoy/envoy.yaml -p 9002:9000 -p 10002:10000 envoy_test

curl -v -d '{"name":"test","endpoints":[{"host":"host.docker.internal", "port":28080}]}' -H "Content-Type: application/json" -X POST http://localhost:18080/upsert_cluster
curl -v -d '{"vhost":"test","cluster":"test"}' -H "Content-Type: application/json" -X POST http://localhost:18080/upsert_route
curl -H 'Host: test' -v http://127.0.0.1:10001/

Setup
1 mysql
2 compass instances
2 envoy instances
2 slice backends
1 lost backend

Scenarios

1, access a non-existing vhost on both envoy instances, should proxy to lost
2, create a cluster, create a vhost, access the vhost on both envoy instances, should get responses
3, get the vhost from admin, should get response
4, delete the vhost, access the host on both instances, should proxy to lost

unreliable restart
client ip
restricted IP
calander redirect
