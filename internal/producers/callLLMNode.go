package producers

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sshfz/consumer-service-substrate/cmd/app/mq"
	"github.com/streadway/amqp"
)

var exchangeName = "dev.topic.spinrequest"

// func CallLLMNode(prompt string, routingKey string) (map[string]interface{}, error) {
// 	ch := mq.GetChannel()

// 	replyQueue, err := ch.QueueDeclare(
// 		"replyQue", // random name
// 		false,      // durable
// 		true,       // delete when unused
// 		false,      // exclusive
// 		false,      // no-wait
// 		nil,        // args
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	bindingKey := "spin.llmreply" // Listen to routing keys like spin.create, spin.delete, etc.
// 	err = ch.QueueBind(
// 		replyQueue.Name,
// 		bindingKey,
// 		exchangeName,
// 		false,
// 		nil,
// 	)

// 	if err != nil {
// 		log.Println(err)
// 	}
// 	corrID := randomString()

// 	msgs, err := ch.Consume(
// 		replyQueue.Name,
// 		corrID,
// 		true,  // auto-ack
// 		false, // exclusive
// 		false, // no-local
// 		false, // no-wait
// 		nil,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = ch.Publish(
// 		exchangeName, // exchange
// 		routingKey,   // routing key (e.g., "spin.create")
// 		false,        // mandatory
// 		false,        // immediate
// 		amqp.Publishing{
// 			ContentType:   "application/json",
// 			CorrelationId: corrID,
// 			ReplyTo:       replyQueue.Name,
// 			Body:          []byte(prompt),
// 		},
// 	)
// 	if err != nil {
// 		log.Printf("❌ Failed to publish message: %v", err)
// 		return nil, err
// 	}

// 	type ResponseStruct struct {
// 		Status string                 `json:"status"`
// 		Code   map[string]interface{} `json:"code"`
// 	}

// 	var responseStruct ResponseStruct

// 	for msg := range msgs {
// 		if msg.CorrelationId == corrID {
// 			// var data map[string]interface{}
// 			err := json.Unmarshal(msg.Body, &responseStruct)
// 			if err != nil {
// 				log.Println(err)
// 				return nil, err
// 			}

// 			log.Println(responseStruct)
// 			if responseStruct.Status == "finished" {
// 				return responseStruct.Code, nil
// 			}
// 			_ = ch.Cancel(corrID, false) // stop this consumer
// 			break
// 		}
// 	}
// 	return responseStruct.Code, nil
// }

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
		consumerTag, // Use proper consumer tag, not correlation ID
		false,       // manual ack - important for reliability
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start consumer: %w", err)
	}

	// Ensure consumer is cancelled when function exits
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

				if responseStruct.Status == "finished" {
					return responseStruct.Code, nil
				}

				// If status is not "finished", continue waiting for more messages
				// You might want to handle other statuses here
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
