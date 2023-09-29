all: azx

azx: *.go
	GO111MODULE=off go build -o $@ *.go

test: azx
	rm -rf .azx
	./azx init -f
	./azx add aca-app -n poc --image duglin/echo --environment demo
