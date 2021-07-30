FROM golang:1.16-alpine
WORKDIR /app
COPY *.go ./
ARG GO111MODULE=off
RUN go build -o /server
EXPOSE 8888
CMD ["/server"]