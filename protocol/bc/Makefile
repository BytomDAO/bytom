all:
	protoc -I/usr/local/include -I. \
      -I${GOPATH}/src \
      --go_out=plugins=grpc:. \
      ./*.proto
clean:
	rm bc.pb.go