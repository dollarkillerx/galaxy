build:
	CGO_ENABLED=0 GOOS="linux" GOARCH="amd64" go build -o cmd/galaxy cmd/main.go

upx:
	upx cmd/galaxy
