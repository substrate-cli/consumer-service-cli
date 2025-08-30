package mq

import (
	"log"

	"github.com/qdrant/go-client/qdrant"
)

var client *qdrant.Client

func SetQDrantClient(conn *qdrant.Client) {
	client = conn
}

func GetQDrantClient() *qdrant.Client {
	if client != nil {
		return client
	}
	log.Fatalf("client has not been initialised")
	return nil
}
