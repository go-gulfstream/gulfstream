
.PHONY: docker-protoc
docker-protoc:
	@docker build -t github.com/go-gulfstream/gulfstream/protoc:latest -f   \
           ./docker/protoc.dockerfile .

.PHONY: proto
proto: docker-protoc
	@bash ./scripts/genproto.bash

.PHONY: mock
mock: docker-mock
	@bash ./scripts/genmock.bash

.PHONY: docker-mock
docker-mock:
	@docker build -t github.com/go-gulfstream/gulfstream/mock:latest -f    \
           ./docker/mockgen.dockerfile .