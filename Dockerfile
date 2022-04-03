FROM alpine:latest
RUN apk update
RUN apk --no-cache add ca-certificates libc-dev libpcap-dev
COPY ./templates /app/templates
COPY ./build/server /app/server
WORKDIR /app
EXPOSE 8080
CMD ["./server"]