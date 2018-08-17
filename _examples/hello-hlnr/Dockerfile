FROM golang:1.10

WORKDIR /go/src/github.com/manifoldco/hello-hlnr
COPY ./hello-hlnr.go ./
COPY ./Makefile ./
COPY ./bin ./
RUN go build -o ./bin/hello-hlnr hello-hlnr.go 

EXPOSE 8080
ENTRYPOINT [ "./bin/hello-hlnr" ]
