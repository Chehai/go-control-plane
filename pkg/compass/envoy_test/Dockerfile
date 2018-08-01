FROM envoyproxy/envoy:latest

CMD /usr/local/bin/envoy --v2-config-only -l debug -c /etc/envoy/envoy.yaml

EXPOSE 9901 9909

# docker run --rm -v $(pwd)/envoy.yaml:/etc/envoy/envoy.yaml -p 9901:9901 -p 9909:9909 envoy_test
