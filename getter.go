package main

import (
	"context"
	"crypto/tls"
	"net/http/httptrace"
	"time"
	"net/http"
	"fmt"
	"sync"

	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/assembla/cony"
	"github.com/xandout/od-check/htrace"
)

func MustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s missing", key)
	}
	return v
}

var  qname, exchange, routingkey, amqpurl string

var qos, workers int
var startTime time.Time
var handledMsgs = 0
func passMsg(data amqp.Delivery) {

	url := fmt.Sprintf("https://%s", string(data.Body))
	ctr, err := htrace.NewClientTrace(url, "GET")
	if err != nil {
		log.Info(err)
	}
	log.Info(ctr.Influx())
	data.Ack(false)

	handledMsgs = handledMsgs + 1
	if handledMsgs%100 == 0 {
		log.Infof("Handled the %vth message, %v from startup(%v)", handledMsgs, time.Since(startTime).Seconds(), startTime)
	}
}

func consume(cli *cony.Client, wg *sync.WaitGroup) {
	log.Info("Starting worker")
	que := &cony.Queue{
		Name:       qname,
		AutoDelete: false,
	}

	exc := cony.Exchange{
		Name:       exchange,
		Kind:       "direct",
		Durable:    true,
		AutoDelete: false,
	}
	bnd := cony.Binding{
		Queue:    que,
		Exchange: exc,
		Key:      routingkey,
	}
	cli.Declare([]cony.Declaration{
		cony.DeclareQueue(que),
		cony.DeclareExchange(exc),
		cony.DeclareBinding(bnd),
	})


	// Declare and register a consumer
	cns := cony.NewConsumer(
		que,
		cony.Qos(qos),
	)


	
	cli.Consume(cns)
	for cli.Loop() {
		select {
		case msg := <-cns.Deliveries():
			go passMsg(msg)
		case err := <-cns.Errors():
			log.Infof("Consumer error: %v\n", err)
		case err := <-cli.Errors():
			log.Infof("Client error: %v\n", err)
		}
	}
	wg.Done()
}


func main() {
	startTime = time.Now()
	amqpurl = MustGetEnv("AMQP_URL")
	qname = MustGetEnv("Q_NAME")
	exchange = MustGetEnv("EXCHANGE_NAME")
	routingkey = MustGetEnv("ROUTING_KEY")
	_qos, qerr := strconv.Atoi(MustGetEnv("QOS"))
	qos = _qos
	if qerr != nil {
		log.Fatal(qerr)
	}
	workernum, werr := strconv.Atoi(MustGetEnv("MAX_WORKERS"))
	if werr != nil {
		log.Fatal(werr)
	}

	var wg sync.WaitGroup
	cli := cony.NewClient(
		cony.URL(amqpurl),
		cony.Backoff(cony.DefaultBackoff),
	)
	if werr != nil {
		log.Fatal(werr)
	}

	for i := 1; i <= workernum; i++ {
		wg.Add(1)
		go consume(cli, &wg)
	}

	wg.Wait()

}
