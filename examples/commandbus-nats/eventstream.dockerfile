FROM scratch

COPY ./bin/eventstream /app/
WORKDIR /app

ENTRYPOINT ["./eventstream"]