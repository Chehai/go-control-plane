`mysql -uroot -e 'TRUNCATE compass.routes'`
`mysql -uroot -e 'TRUNCATE compass.clusters'`

admin_interface = "http://localhost:18080"
slices = 100
vhosts_per_slice = 300

slices.times do |i|
  start = Time.now
  out = `curl -s -o /dev/null -w "%{http_code}" -d '{"name":"test#{i}","endpoints":[{"host":"192.168.65.2", "port":28080}]}' -H "Content-Type: application/json" -X POST #{admin_interface}/upsert_cluster`
  puts "creating cluster test#{i}: #{out.chomp} : #{Time.now - start}"
end

(slices * vhosts_per_slice).times do |i|
  start = Time.now
  out = `curl -s -o /dev/null -w "%{http_code}" -d '{"vhost":"test#{i}","cluster":"test#{i % 100}"}' -H "Content-Type: application/json" -X POST #{admin_interface}/upsert_route`
  puts "creating route test#{i}: #{out.chomp} : #{Time.now - start}"
end
