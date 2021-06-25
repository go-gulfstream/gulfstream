FROM scratch

COPY ./bin/client /app/
WORKDIR /app

ENTRYPOINT ["./client"]