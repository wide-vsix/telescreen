FROM golang:latest

WORKDIR /go/src/github.com/wide-vsix/dns-query-interceptor
COPY . .
RUN chmod +x ./build/wait-for-postgres.sh

RUN go install github.com/wide-vsix/dns-query-intereceptor/interceptor
RUN GOOS=linux GOARCH=amd64 go build -o interceptor ./interceptor.go

CMD ["./build/wait-for-postgres.sh", "./interceptor"]