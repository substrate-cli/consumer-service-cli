package connections

import (
	"log"

	"github.com/substrate-cli/consumer-service-cli/cmd/app/mq"
	"github.com/substrate-cli/consumer-service-cli/internal/consumers"
	"github.com/substrate-cli/consumer-service-cli/internal/utils"
	"github.com/streadway/amqp"
)

// StartConsumer sets up the RabbitMQ topic consumer
func StartConsumer() {
	// Connect to RabbitMQ
	conn, err := amqp.Dial(utils.GetAMQPUrl())
	// conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("❌ Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	mq.SetChannel(ch)
	if err != nil {
		log.Fatalf("❌ Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// 1. Declare the topic exchange
	exchangeName := "dev.topic.spinrequest"
	err = ch.ExchangeDeclare(
		exchangeName,
		"topic", // exchange type
		true,    // durable
		false,   // auto-delete
		false,   // internal
		false,   // no-wait
		nil,     // args
	)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange: %v", err)
	}

	// 2. Declare a queue
	queueName := "spin_consumer_queue"
	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Fatalf("❌ Failed to declare queue: %v", err)
	}

	// 3. Bind the queue to the topic exchange
	bindingKey := "spin.*" // Listen to routing keys like spin.create, spin.delete, etc.
	err = ch.QueueBind(
		q.Name,
		bindingKey,
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("❌ Failed to bind queue: %v", err)
	}

	// 4. Start consuming messages
	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer tag
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Fatalf("❌ Failed to register a consumer: %v", err)
	}

	log.Println("🟢 Listening for topic messages on 'dev.topic.spinrequest'...")

	var maxWorkers = 5
	sem := make(chan struct{}, maxWorkers)

	for msg := range msgs {
		sem <- struct{}{} // acquire slot
		log.Printf("📥 [%s] %s", msg.RoutingKey, msg.Body)
		go func(m amqp.Delivery) {
			defer func() { <-sem }() // release slot

			log.Printf("📥 [%s] %s", m.RoutingKey, m.Body)

			switch m.RoutingKey {
			case "spin.create":
				err := consumers.HandleSpinConsumer(m.Body)
				if err != nil {
					log.Println("❌ SpinRequest failed:", err)
					return
				}
			default:
				log.Printf("⚠️ Unknown topic: %s", m.RoutingKey)
			}
		}(msg)
	}
}
