FROM golang:1.20
WORKDIR /source
# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod ./
RUN go mod download && go mod verify
COPY main.go main.go
RUN go build -v -o /app/dedupe .

CMD ["/app/dedupe"]
