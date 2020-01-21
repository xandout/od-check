FROM golang

WORKDIR /go/src/github.com/xandout/od-check
COPY . ./

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["od-check"]