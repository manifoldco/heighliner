FROM manifoldco/scratch-certificates
USER 7171:8787

ARG BINARY

COPY  $BINARY ./controller
ENTRYPOINT ["./controller"]
