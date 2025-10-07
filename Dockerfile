Final Stage

FROM debian:bookworm

EXPOSE 8080

RUN apt-get update && apt-get install -y --no-install-recommends 

libsqlite3-0 

&& rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /webapi ./webapi

CMD ["./webapi"]