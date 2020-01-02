FROM golang

WORKDIR /go/src/app
COPY main.go ./main.go
COPY million.csv .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["app"]