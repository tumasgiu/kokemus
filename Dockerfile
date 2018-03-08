FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY ./templates /app/templates
COPY ./build/server /app/server
WORKDIR /app
CMD ["./server"]