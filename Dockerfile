FROM golang:1.11

ARG BINARY

WORKDIR /go/src/github.com/manifoldco/heighliner
COPY . ./

RUN make bootstrap
RUN make vendor
RUN make $BINARY
RUN cp $BINARY /controller

FROM manifoldco/scratch-certificates
USER 7171:8787

COPY --from=0 /controller /controller
ENTRYPOINT ["/controller"]
