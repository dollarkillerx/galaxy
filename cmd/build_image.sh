#!/bin/bash

source_path=./cmd
go_file=main.go
image_name=galaxy
build_output=galaxy

CGO_ENABLED=0 GOOS="linux" GOARCH="amd64" go build -o $source_path/$build_output $source_path/$go_file

docker rmi -f $image_name
docker build -f $source_path/Dockerfile -t $image_name  .
rm $source_path/$build_output
docker save -o $image_name.tar $image_name:latest
