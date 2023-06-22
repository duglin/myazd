all: azd

azd: azd.go
	GO111MODULE=off go build -o $@ *.go

test: azd
	./azd apply acaRedis.json acaApp.json
