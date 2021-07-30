FROM golang:1.16-alpine
WORKDIR /app
COPY *.go ./
ARG GO111MODULE=off
RUN go build -o /client
CMD ["/client"]