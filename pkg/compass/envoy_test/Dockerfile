FROM envoyproxy/envoy

CMD /usr/local/bin/envoy --v2-config-only -l debug -c /etc/envoy/envoy.yaml

EXPOSE 9901 10000

# docker run --rm -v $(pwd)/envoy.yaml:/etc/envoy/envoy.yaml -p 9901:9901 -p 10000:10000 envoy_test
