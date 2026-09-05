#!/bin/sh
# Runs each route of the server and writes the answer. The port stands in $1.
# The output holds no time and no random value, thus a diff of 2 runs proves
# that the 2 servers agree.
set -e
port=$1
h="Authorization: Bearer  e1c37dfd973353b78bb71df050e2c6e72d53034e148920383968ae49b96f1fd2"
base="http://127.0.0.1:$port"
say() { printf '\n=== %s ===\n' "$1"; }

say "login ok"
curl -s -w '\nstatus %{http_code}\n' -X POST "$base/login" \
  -d '{"email":"email@example.com","password":"password"}'
say "login wrong"
curl -s -w '\nstatus %{http_code}\n' -X POST "$base/login" -d '{"email":"x","password":"y"}'
say "no token"
curl -s -w '\nstatus %{http_code}\n' "$base/user"
say "user"
curl -s -w '\nstatus %{http_code}\n' -H "$h" "$base/user"
say "continent"
curl -s -w '\nstatus %{http_code}\n' -H "$h" "$base/continent"
say "destination count"
curl -s -H "$h" "$base/destination" | wc -c
say "destination first"
curl -s -H "$h" "$base/destination" | head -c 400
say "activity of alaska"
curl -s -H "$h" "$base/destination/alaska/activity" | wc -c
say "activity of kyoto first"
curl -s -H "$h" "$base/destination/kyoto/activity" | head -c 300
say "booking list"
curl -s -w '\nstatus %{http_code}\n' -H "$h" "$base/booking" | head -c 600
say "booking 0"
curl -s -w '\nstatus %{http_code}\n' -H "$h" "$base/booking/0" | head -c 400
say "booking new"
curl -s -w '\nstatus %{http_code}\n' -X POST -H "$h" "$base/booking" \
  -d '{"id":null,"startDate":"2024-09-01T00:00:00Z","endDate":"2024-09-08T00:00:00Z","name":"Kyoto, Asia","destinationRef":"kyoto","activitiesRef":["a1"]}'
say "booking with id"
curl -s -w '\nstatus %{http_code}\n' -X POST -H "$h" "$base/booking" \
  -d '{"id":7,"startDate":"2024-09-01T00:00:00Z","endDate":"2024-09-08T00:00:00Z","name":"x","destinationRef":"kyoto","activitiesRef":[]}'
say "booking 1 after the post"
curl -s -w '\nstatus %{http_code}\n' -H "$h" "$base/booking/1" | head -c 300
say "delete 1"
curl -s -w '\nstatus %{http_code}\n' -X DELETE -H "$h" "$base/booking/1"
say "booking 1 after the delete"
curl -s -w '\nstatus %{http_code}\n' -H "$h" "$base/booking/1"
say "booking 99"
curl -s -w '\nstatus %{http_code}\n' -H "$h" "$base/booking/99"
