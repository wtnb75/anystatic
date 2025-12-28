#! /bin/sh

hosts="bencha bencht benchn"
files="small.txt medium.txt large.txt"
paramc="2 3 4 5 6 8 10 15 20 30 40 50"
duration=30s
output=results
mkdir -p $output

# single thread, single connection
for host in $hosts; do
for file in $files; do
  [ -f $output/single.$host.$file.nocomp ] || \
  wrk -t1 -c1 -d$duration --latency http://$host/$file > $output/single.$host.$file.nocomp
  [ -f $output/single.$host.$file.gzip ] || \
  wrk -H "Accept-Encoding: gzip" -t1 -c1 -d$duration --latency http://$host/$file > $output/single.$host.$file.gzip
  [ -f $output/single.$host.$file.any ] || \
  wrk -H "Accept-Encoding: gzip, br, zstd" -t1 -c1 -d$duration --latency http://$host/$file > $output/single.$host.$file.any
done
done

# not compressed
for host in $hosts; do
for file in $files; do
for c in $paramc; do
  [ -f $output/$host.$c.$file.nocomp ] || \
  wrk -t2 -c$c -d$duration --latency http://$host/$file > $output/$host.$c.$file.nocomp
done
done
done

# compressed
for host in $hosts; do
for file in $files; do
for c in $paramc; do
  [ -f $output/$host.$c.$file.gzip ] || \
  wrk -H "Accept-Encoding: gzip" -t2 -c$c -d$duration --latency http://$host/$file > $output/$host.$c.$file.gzip
done
done
done

# compressed2
for host in $hosts; do
for file in $files; do
for c in $paramc; do
  [ -f $output/$host.$c.$file.any ] || \
  wrk -H "Accept-Encoding: gzip, br, zstd" -t2 -c$c -d$duration --latency http://$host/$file > $output/$host.$c.$file.any
done
done
done
