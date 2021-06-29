#!/bin/bash
path="$(pwd)"
declare -a mocks=(
"stream/storage"
"stream/eventbus"
"stream/mutator"
"stream/state"
"stream/journal"
)
for i in "${mocks[@]}"
do
   parts=($(echo $i | tr '/' "\n"))
   index=$((${#parts[@]}-2))
   pkg="${parts[index]}"
   docker run --rm -v $path:$path github.com/go-gulfstream/gulfstream/mock -package="mock$pkg" -destination="$path/mocks/${i}.go" -source="$path/pkg/${i}.go"
done;