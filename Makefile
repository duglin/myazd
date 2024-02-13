APP=azx

all: ${APP}

${APP} ${APP}-mac ${APP}-linux ${APP}-win.exe: *.go
	GO111MODULE=off GOOS=darwin  GOARCH=amd64 go build -o ${APP}-mac *.go
	GO111MODULE=off GOOS=linux   GOARCH=amd64 go build -o ${APP}-linux *.go
	GO111MODULE=off GOOS=windows GOARCH=amd64 go build -o ${APP}-win.exe *.go
	ln -fs ./${APP}-mac ./${APP}

test: ${APP}
	rm -rf .${APP}       # erase generated/config files
	./${APP} init -f     # recreate them
	./${APP} add aca-app -n poc --image duglin/echo --environment demo

package: .package
.package: ${APP}
	rm -f demo-${APP}.tar
	tar -cf demo-${APP}.tar ${APP} ${APP}-* .demoscript demo1 demo2
	cp demo-${APP}.tar "${HOME}/OneDrive - Microsoft/public/"
	touch .package

clean:
	rm -f ${APP} ${APP}-mac ${APP}-linux ${APP}-win.exe
	rm -rf .${APP} .package
