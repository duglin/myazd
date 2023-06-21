all: azd

azd: azd.go
	GO111MODULE=off go build -o $@ azd.go jsonc.go

test: azd
	./azd apply acaRedis.json acaApp.json
