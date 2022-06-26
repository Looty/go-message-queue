package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

type Message struct {
	text      string
	timestamp string
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
	router.LoadHTMLGlob("templates/**")

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
	maxMessages := viperEnvVariable("MAXMESSAGES")
	rmqUser := viperEnvVariable("RMQUSER")
	rmqPass := viperEnvVariable("RMQPASS")
	rmqAddr := viperEnvVariable("RMQADDR")
	rmqPort := viperEnvVariable("RMQPORT")

	rmqConnectionString := fmt.Sprintf("amqp://%s:%s@%s:%s/", rmqUser, rmqPass, rmqAddr, rmqPort)
	log.Printf("[INFO] Searcing for RMQ server %s...", rmqConnectionString)
	conn, err := amqp.Dial(rmqConnectionString)
	handleError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	handleError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	handleError(err, "Failed to declare a queue")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":    "go-message-queue",
			"maxInput": maxMessages,
		})
	})

	router.POST("/post", func(c *gin.Context) {
		text := c.PostForm("messageText")
		timestamp := fmt.Sprintf("%s", time.Now())
		quantity := c.PostForm("messageQuantity")

		quantityToInt, err := strconv.Atoi(quantity)
		if err != nil {
			log.Fatalf("Error while converting quantity string to int %s", err)
		}

		messageArray := make([]Message, quantityToInt)

		for i := range messageArray {
			messageArray[i] = Message{text, timestamp}

			err = ch.Publish(
				"",     // exchange
				q.Name, // routing key
				false,  // mandatory
				false,  // immediate
				amqp.Publishing{
					ContentType: "text/plain",
					Body:        []byte(messageArray[i].text),
				})
			handleError(err, "Failed to publish a message")
			log.Printf(" [x] Sent %s\n", messageArray[i].text)
		}

		// post to MQM
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	router.Run(":" + appPort)
}
