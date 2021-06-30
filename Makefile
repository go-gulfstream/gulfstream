PROJECT_PATH = $(PWD)
DEPLOYMENTS_LOCAL = $(PROJECT_PATH)/deployments/local
DEPLOYMENTS_LOCAL_RUN = @docker-compose                               \
      -f $(DEPLOYMENTS_LOCAL)/network.yml                             \
      -f $(DEPLOYMENTS_LOCAL)/kafka.yml                               \
      -f $(DEPLOYMENTS_LOCAL)/postgres.yml                            \

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

.PHONY: test
test:
	@rm -rf $(DEPLOYMENTS_LOCAL)/tmp
	@mkdir $(DEPLOYMENTS_LOCAL)/tmp
	@$(DEPLOYMENTS_LOCAL_RUN)  up -d
	@sleep 2
	@GULFSTREAM_INTEGRATION=ok go test `go list ./... | grep tests`; exit 0;
	@go test `go list ./... | grep pkg`; exit 0;
	@$(DEPLOYMENTS_LOCAL_RUN) down --volumes
