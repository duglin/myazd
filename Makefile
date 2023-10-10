APP=azx

all: ${APP}

${APP} ${APP}.mac ${APP}.linux ${APP}.exe: *.go
	GO111MODULE=off GOOS=darwin  GOARCH=amd64 go build -o ${APP}.mac *.go
	GO111MODULE=off GOOS=linux   GOARCH=amd64 go build -o ${APP}.linux *.go
	GO111MODULE=off GOOS=windows GOARCH=amd64 go build -o ${APP}.exe *.go
	ln -s ./${APP}.mac ./${APP}

test: ${APP}
	rm -rf .${APP}
	./${APP} init -f
	./${APP} add aca-app -n poc --image duglin/echo --environment demo

clean:
	rm -f ${APP} ${APP}.exe
