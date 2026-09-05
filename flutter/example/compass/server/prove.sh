#!/bin/sh
# Runs the server 2 times, first with go run and then with the Dart output of
# gart, and it diffs the 2 answers. The diff is the proof that the compiler
# keeps the meaning of the program.
set -e
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../../../.." && pwd)
out=${TMPDIR:-/tmp}
cd "$here"

# wait_port polls the login route until the server answers.
wait_port() {
	i=0
	while [ $i -lt 120 ]; do
		if curl -s -o /dev/null -X POST "http://127.0.0.1:$1/login" -d '{}'; then
			return 0
		fi
		sleep 1
		i=$((i + 1))
	done
	echo "the server on the port $1 gave no answer"
	exit 1
}

go build -o "$out/compassserver" .
"$out/compassserver" -assets "$here/../assets" -port 8091 &
gopid=$!
trap 'kill $gopid 2>/dev/null || true' EXIT
wait_port 8091
sh requests.sh 8091 > "$out/go.txt"
kill $gopid 2>/dev/null || true

"$root/build/gart" build -o "$here/gen" -runtime "$root/runtime/goruntime" ./...
(cd gen && dart pub get > /dev/null && dart run bin/main.dart -assets "$here/../assets" -port 8092) &
dartpid=$!
trap 'kill $gopid $dartpid 2>/dev/null || true' EXIT
wait_port 8092
sh requests.sh 8092 > "$out/dart.txt"
kill $dartpid 2>/dev/null || true

if diff -u "$out/go.txt" "$out/dart.txt"; then
	echo "the 2 servers agree"
else
	echo "the 2 servers differ"
	exit 1
fi
