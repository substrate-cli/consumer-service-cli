package main

import (
	"log"
	"time"
)

func startHeartbeat(stopChan <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Minute)
	log.Printf("heartbeat ping - consumer-service", time.Now().Format(time.RFC3339))

	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("heartbeat ping - consumer-service", time.Now().Format(time.RFC3339))
			case <-stopChan:
				log.Println("Stopping heartbeat...")
				ticker.Stop()
				return
			}
		}
	}()
}
