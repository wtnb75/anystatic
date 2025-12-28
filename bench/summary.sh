#! /bin/sh

hosts="bencha bencht benchn"
files="small.txt medium.txt large.txt"
paramc="2 3 4 5 6 8 10 15 20 30 40 50"
output=results

do1(){
  [ -f "$1" ] || return
  gawk '
($1=="99%"){
  match($2, /([0-9\.]*)(ms|us)/, arr);
  if(arr[2]=="ms"){
    lat=arr[1]*1000;
  }else if(arr[2]=="us"){
    lat=arr[1];
  }
}
($1=="Requests/sec:"){
  reqs=$2;
  print reqs, lat;
  lat=0;
}' $*
}

for mode in nocomp gzip any; do
for host in $hosts; do
for file in $files; do
  [ -f $output/single.$host.$file.$mode ] && \
  do1 $output/single.$host.$file.$mode > $output/summary.$host.$file.$mode
  for c in $paramc ; do
    [ -f $output/$host.$c.$file.$mode ] && \
    do1 $output/$host.$c.$file.$mode >> $output/summary.$host.$file.$mode
  done
done
done
done
