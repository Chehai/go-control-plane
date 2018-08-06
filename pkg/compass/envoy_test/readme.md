docker build -t envoy_test .
docker run --rm -p 28080:80 jmalloc/echo-server
docker run --rm -v $(pwd)/envoy.yaml:/etc/envoy/envoy.yaml -p 9001:9000 -p 10001:10000 envoy_test
docker run --rm -v $(pwd)/envoy.yaml:/etc/envoy/envoy.yaml -p 9002:9000 -p 10002:10000 envoy_test

curl -v -d '{"name":"test","endpoints":[{"host":"192.168.65.2", "port":28080}]}' -H "Content-Type: application/json" -X POST http://localhost:18080/upsert_cluster
curl -v -d '{"vhost":"test","cluster":"test"}' -H "Content-Type: application/json" -X POST http://localhost:18080/upsert_route
curl -H 'Host: test' -v http://127.0.0.1:10001/
