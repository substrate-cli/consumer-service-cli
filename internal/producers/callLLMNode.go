package producers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"github.com/substrate-cli/consumer-service-cli/cmd/app/mq"
)

var exchangeName = "dev.topic.spinrequest"

func CallLLMNode(prompt string, routingKey string) (map[string]interface{}, error) {
	ch := mq.GetChannel()

	// Generate unique names for this request
	corrID := randomString()
	replyQueueName := fmt.Sprintf("reply_%s", corrID) // Unique queue name
	consumerTag := fmt.Sprintf("consumer_%s", corrID) // Unique consumer tag

	// Declare reply queue
	replyQueue, err := ch.QueueDeclare(
		replyQueueName, // unique name
		false,          // durable
		true,           // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare reply queue: %w", err)
	}

	// Bind queue to exchange
	bindingKey := "spin.llmreply"
	err = ch.QueueBind(
		replyQueue.Name,
		bindingKey,
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	// Start consuming before publishing (important!)
	msgs, err := ch.Consume(
		replyQueue.Name,
		consumerTag, // Using proper consumer tag
		false,       // manual ack - important for reliability
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start consumer: %w", err)
	}

	defer func() {
		if err := ch.Cancel(consumerTag, false); err != nil {
			log.Printf("Failed to cancel consumer: %v", err)
		}
	}()

	// Publish the message
	err = ch.Publish(
		exchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: corrID,
			ReplyTo:       replyQueue.Name,
			Body:          []byte(prompt),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("✅ Message sent with correlation ID: %s", corrID)
	log.Printf("🔄 Waiting for reply on queue: %s", replyQueue.Name)
	log.Println("Routing key => ", routingKey)

	// Set up timeout
	timeout := time.After(15 * time.Minute) // Adjust timeout as needed

	type ResponseStruct struct {
		Status string                 `json:"status"`
		Code   map[string]interface{} `json:"code"`
	}

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return nil, fmt.Errorf("consumer channel closed")
			}

			log.Printf("📨 Received message with correlation ID: %s", msg.CorrelationId)

			// Check correlation ID
			if msg.CorrelationId == corrID {
				var responseStruct ResponseStruct
				err := json.Unmarshal(msg.Body, &responseStruct)
				if err != nil {
					msg.Nack(false, false) // Reject the message
					return nil, fmt.Errorf("failed to unmarshal response: %w", err)
				}

				log.Printf("✅ Received response: %+v", responseStruct)

				// Acknowledge the message
				msg.Ack(false)

				if responseStruct.Status == "failed" {
					return nil, errors.New("unable to create cluster")
				}

				if responseStruct.Code == nil {
					return nil, errors.New("Undefined code structure")
				}

				if responseStruct.Status == "finished" {
					return responseStruct.Code, nil
				}

			} else {
				// Wrong correlation ID, reject and continue
				log.Printf("❌ Correlation ID mismatch. Expected: %s, Got: %s", corrID, msg.CorrelationId)
				msg.Nack(false, true) // Reject and requeue
			}

		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for reply after 15 minutes")
		}
	}
}
func randomString() string {
	return uuid.NewString()
}
