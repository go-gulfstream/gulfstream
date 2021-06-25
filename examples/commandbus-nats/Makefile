.PHONY: example
example:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o ./bin/client ./client/*.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o ./bin/eventstream ./eventstream/*.go
	docker-compose up --build

