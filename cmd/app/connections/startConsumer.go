package connections

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/sshfz/consumer-service-substrate/cmd/app/mq"
	"github.com/sshfz/consumer-service-substrate/internal/consumers"
	"github.com/sshfz/consumer-service-substrate/internal/helpers"
	"github.com/sshfz/consumer-service-substrate/internal/utils"
	"github.com/sshfz/consumer-service-substrate/internal/webhooks"
	"github.com/streadway/amqp"
)

// StartConsumer sets up the RabbitMQ topic consumer
func StartConsumer() {
	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
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
				err := handleSpinConsumer(m.Body)
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

// handleTask processes the received task message
func handleSpinConsumer(body []byte) error {

	log.Printf("inside handle-spin-consumer")

	var payload consumers.SpinRequest
	err := json.Unmarshal(body, &payload)
	if err != nil {
		log.Println("failed to decode json")
		return err
	}
	log.Println("User prompt => ", payload.Prompt)
	if utils.GetMode() == "cli" {
		utils.SetCLIApiKey(payload.ApiKey)
	}
	//precheck -------
	type Response struct {
		Is_valid_prompt  bool
		Response         string
		Reason           string
		Requires_backend bool
	}
	var response Response
	res, err := helpers.CallAnthropicPrecheck(payload.Prompt)
	if err != nil {
		log.Println("Error in anthropic precheck.")
		log.Println(err)
		return err
	}
	err = json.Unmarshal([]byte(res), &response)
	if err != nil {
		log.Println("Error in decoding anthropic precheck response", err)
		return err
	}
	if !response.Is_valid_prompt {
		///call webhook in api-server for failed attempt
		err := webhooks.PrecheckAction("failed", response.Reason)
		if err != nil {
			log.Println("api-service webhook failed")
			return err
		}
		log.Println("invalid prompt detected for app generation, proceeding to call webhook in api-server")
		return errors.New("invalid prompt detected for app generation")
	}
	///calling webhook for successful precheck -----
	log.Println("Anthropic Precheck passed, proceeding for code generation...")

	err = webhooks.PrecheckAction("finished", response.Response)
	if err != nil {
		log.Println("api-service webhook failed")
		return err
	}
	log.Println("Proceeding for code generation...")
	log.Println("Initiating code generation for => ", payload.Prompt)

	if !response.Requires_backend {
		log.Println("Backend not required for cluster")
		log.Println("Proceeding to generate next js code generation.")
		err = consumers.SpinRequestConsumer(payload)
	} else {
		log.Println("Backend required for cluster")
		log.Println("Proceeding to generate full stack application")
		///generating backend prompt -------
		backendStructPrompt, err := helpers.CallAnthropicConstructBackendPrompt(payload.Prompt)
		if err != nil {
			log.Println("Error constrcuting backend prompt")
			return err
		}
		err = webhooks.PrecheckAction("finished", "backend prompt generated.")
		if err != nil {
			log.Println("api-service webhook failed")
			return err
		}
		log.Println("Backend Struct Prompt => ", backendStructPrompt)
		payload.BackendPrompt = backendStructPrompt
		err = consumers.SpinRequestConsumerFullStack(payload)
	}

	if err != nil {
		log.Println("error while spinning up request.")
		return err
	}
	log.Printf("🔧 Processing task: %s", string(body))
	return nil
}
