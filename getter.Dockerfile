FROM golang

WORKDIR /go/src/app
COPY getter.go ./main.go

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["app"]