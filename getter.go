package main

import (
	"context"
	"strconv"
	"fmt"
	"crypto/tls"
	"net/http"
	"time"
	"net/http/httptrace"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/assembla/cony"

)

func MustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s missing", key)
	}
	return v
}

func passMsg(data amqp.Delivery) {
	url := fmt.Sprintf("https://%s", string(data.Body))
	req, _ := http.NewRequest("HEAD", url, nil)

	
    var start, connect, dns, tlsHandshake time.Time

	var startDur, connectDur, dnsDur, tlsDur time.Duration
    trace := &httptrace.ClientTrace{
        DNSStart: func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
        DNSDone: func(ddi httptrace.DNSDoneInfo) {
            dnsDur = time.Since(dns)
			log.Infof("DNS Done: %v\n", dnsDur)
        },

        TLSHandshakeStart: func() { tlsHandshake = time.Now() },
        TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			tlsDur = time.Since(tlsHandshake)
            log.Infof("TLS Handshake: %v\n", tlsDur)
        },

        ConnectStart: func(network, addr string) { connect = time.Now() },
        ConnectDone: func(network, addr string, err error) {
			connectDur = time.Since(connect)
            log.Infof("Connect time: %v\n", connectDur)
        },

        GotFirstResponseByte: func() {
			startDur = time.Since(start)
            log.Infof("Time from start to first byte: %v\n", startDur)
        },
    }
	
	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()
    req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace)).WithContext(ctx)
    start = time.Now()
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Info(err)
	}
	log.Info(resp)
    
	data.Ack(false)

}

func main() {

	// Construct new client with the flag url
	// and default backoff policy
	cli := cony.NewClient(
		cony.URL("amqp://user:bitnami@rabbitmq:5672/"),
		cony.Backoff(cony.DefaultBackoff),
	)

	// Declarations
	// The queue name will be supplied by the AMQP server
	que := &cony.Queue{
		Name:       MustGetEnv("Q_NAME"),
		AutoDelete: false,
	}

	exc := cony.Exchange{
		Name:       MustGetEnv("EXCHANGE_NAME"),
		Kind:       "direct",
		Durable:    true,
		AutoDelete: false,
	}
	bnd := cony.Binding{
		Queue:    que,
		Exchange: exc,
		Key:      MustGetEnv("ROUTING_KEY"),
	}
	cli.Declare([]cony.Declaration{
		cony.DeclareQueue(que),
		cony.DeclareExchange(exc),
		cony.DeclareBinding(bnd),
	})

	qos, err := strconv.Atoi(MustGetEnv("QOS"))
	if err != nil {
		log.Fatal("QOS is not a number")
	}

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
			// If when we built the consumer we didn't use
			// the "cony.AutoAck()" option this is where we'd
			// have to call the "amqp.Deliveries" methods "Ack",
			// "Nack", "Reject"
			//
			// msg.Ack(false)
			// msg.Nack(false)
			// msg.Reject(false)
		case err := <-cns.Errors():
			log.Infof("Consumer error: %v\n", err)
		case err := <-cli.Errors():
			log.Infof("Client error: %v\n", err)
		}
	}
}
