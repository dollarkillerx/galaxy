build:
	CGO_ENABLED=0 GOOS="linux" GOARCH="amd64" go build -o cmd/galaxy cmd/main.go

build_docker_images:
	bash cmd/build_image.sh

upx:
	upx cmd/galaxy

run_test:
	go run cmd/main.go
