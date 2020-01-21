package main

import (
	"bufio"
	"encoding/csv"
	"io"
	"math/rand"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/assembla/cony"
	"github.com/streadway/amqp"
)

func main() {

	// Construct new client with the flag url
	// and default backoff policy
	cli := cony.NewClient(
		cony.URL("amqp://user:bitnami@rabbitmq:5672/"),
		cony.Backoff(cony.DefaultBackoff),
	)

	// Declare the exchange we'll be using
	exc := cony.Exchange{
		Name:       "domains",
		Kind:       "direct",
		AutoDelete: false,
		Durable:    true,
	}
	cli.Declare([]cony.Declaration{
		cony.DeclareExchange(exc),
	})

	// Declare and register a publisher
	// with the cony client.
	// This needs to be "global" per client
	// and we'll need to use this exact value in
	// our handlers (contexts should be of help)
	pbl := cony.NewPublisher(exc.Name, "")
	cli.Publish(pbl)

	// Start our loop in a new gorouting
	// so we don't block this one

	go func() {
		for cli.Loop() {
			select {
			case err := <-cli.Errors():
				log.Errorf("Got err: %v", err)
			case blocked := <-cli.Blocking():
				log.Errorf("Client is blocked %v\n", blocked)

			}
		}
	}()

	log.Infof("Starting")
	csvFile, err := os.Open("million.csv")
	if err != nil {
		log.Fatal(err)
	}
	reader := csv.NewReader(bufio.NewReader(csvFile))
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator

	i := 0
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		i = i + 1

		region := [...]string{"us-east-1", "us-east-2", "us-west-2"}[rand.Intn(3)]
		region = "us-east-1"
		if i%100 == 0 {
			log.Infof("Added item %v to %v", i, region)
		}
		

		err := pbl.PublishWithRoutingKey(amqp.Publishing{
			Body: []byte(line[2]),
		}, region)
		if err != nil {
			log.Infof("Got an AMQP error: %v", err)
		}

	}
	log.Infof("Added %v records total", i)
	cli.Close()
}
