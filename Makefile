
.PHONY: docker-protoc
docker-protoc:
	@docker build -t github.com/go-gulfstream/gulfstream/protoc:latest -f   \
           ./docker/protoc.dockerfile .

.PHONY: proto
proto: docker-protoc
	@bash ./scripts/genproto.bash