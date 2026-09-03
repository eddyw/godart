#!/bin/bash
# fdwatch.sh samples the open file count of the machine, and it names the
# processes that hold the most when the count crosses 60% of the limit.
#
# sysctl needs no file descriptor of its own, thus this script keeps working
# after lsof stops. Run it next to a long build:
#
#     scripts/fdwatch.sh /tmp/fd.log &
#     make ci
set -u
log=${1:-/tmp/fdwatch.log}
limit=$(sysctl -n kern.maxfiles)
alarm=$((limit * 60 / 100))
: > "$log"
while true; do
  n=$(sysctl -n kern.num_files)
  echo "$(date +%H:%M:%S) $n" >> "$log"
  if [ "$n" -gt "$alarm" ]; then
    {
      echo "=== $n of $limit open files ==="
      lsof -n 2>/dev/null | awk '{print $1}' | sort | uniq -c | sort -rn | head -15
      ps -eo pid,etime,command | grep -E "[g]o |[d]art |[f]lutter " | head -30
    } >> "$log"
  fi
  sleep 1
done
