FROM golang:1.10

ARG BINARY

WORKDIR /go/src/github.com/manifoldco/heighliner
COPY . ./

RUN CGO_ENABLED=0 go build -a -i .
RUN cp heighliner /controller

FROM manifoldco/scratch-certificates
USER 7171:8787

COPY --from=0 /controller /controller
ENTRYPOINT ["/controller"]
