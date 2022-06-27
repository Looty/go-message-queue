package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

type Body struct {
	Message string `json:"message" binding:"required"`
}

func viperEnvVariable(key string) string {
	// viper.Get() returns an empty interface{}
	// to get the underlying type of the key,
	// we have to do the type assertion, we know the underlying value is string
	// if we type assert to other type it will throw an error
	value, ok := viper.Get(key).(string)

	// If the type is a string then ok will be true
	// ok will make sure the program not break
	if !ok {
		log.Fatalf("Invalid type assertion")
	}

	return value
}

func handleError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	router := gin.Default()

	// SetConfigFile explicitly defines the path, name and extension of the config file.
	// Viper will use this and not check any of the config paths.
	// .env - It will search for the .env file in the current directory
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Find and read the config file
	err := viper.ReadInConfig()

	if err != nil {
		log.Fatalf("Error while reading config file %s", err)
	}

	appPort := viperEnvVariable("PORT")
	rmqUser := viperEnvVariable("RMQUSER")
	rmqPass := viperEnvVariable("RMQPASS")
	rmqAddr := viperEnvVariable("RMQADDR")
	rmqPort := viperEnvVariable("RMQPORT")
	rmqQueueName := viperEnvVariable("RMQ_QUEUE_NAME")

	rmqConnectionString := fmt.Sprintf("amqp://%s:%s@%s:%s/", rmqUser, rmqPass, rmqAddr, rmqPort)
	log.Printf("[INFO] Searcing for RMQ server %s...", rmqConnectionString)
	conn, err := amqp.Dial(rmqConnectionString)
	handleError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	handleError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		rmqQueueName, // name
		false,        // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	handleError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	handleError(err, "Failed to register a consumer")

	var forever chan struct{}

	go func() {
		log.Printf("Consumer ready, PID: %d", os.Getpid())
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

	router.Run(":" + appPort)
}
