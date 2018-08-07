(100*300).times do |i|
  host = "test#{i}"
  start = Time.now
  res_code = `curl -s -o /dev/null -w "%{http_code}" -H "Host: #{host}" http://127.0.0.1:10001`
  puts "attempt: #{i}: #{host}: res_code: #{res_code}: response_time: #{Time.now - start}"
  #sleep 0.1
end
