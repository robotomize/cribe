package bot

import "github.com/streadway/amqp"

func NewAMQPBroker(connection *amqp.Connection) *AMQPBroker {
	return &AMQPBroker{Connection: connection}
}

type AMQPBroker struct {
	*amqp.Connection
}

func (a *AMQPBroker) Chan() (AMQPChannel, error) {
	return a.Channel()
}
