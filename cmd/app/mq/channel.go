package mq

import "github.com/streadway/amqp"

var channel *amqp.Channel

func SetChannel(chann *amqp.Channel) {
	channel = chann
}

func GetChannel() *amqp.Channel {
	return channel
}
