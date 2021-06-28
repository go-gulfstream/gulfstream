path="$(pwd)"
for proto in $(find . -name *.proto); do
    docker run \
    --rm -v $path:$path -w $path github.com/go-gulfstream/gulfstream/protoc --proto_path=. \
    --gogoslick_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,plugins=grpc:. -I . $proto
done;