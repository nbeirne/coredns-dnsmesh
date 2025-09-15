rm -rf dns
go get
go build
./dns -conf ../docker/Corefile
