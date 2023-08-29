package wsh

import (
	"log"
)

type eBroadcast struct {
	mt     int
	msg    []byte
	topics []string
}

type eSubscribe struct {
	client *client
	topics []string
}

type eUnsubscribe struct {
	client *client
	topics []string
}

type topic struct {
	name    string
	clients map[*client]struct{}
}

func newTopic(name string) *topic {
	return &topic{
		name:    name,
		clients: make(map[*client]struct{}),
	}
}

type Hub struct {
	clients map[*client]struct{}
	topics  map[string]*topic

	chRegister    chan *client
	chDeregister  chan *client
	chSubscribe   chan eSubscribe
	chUnsubscribe chan eUnsubscribe
	chBroadcast   chan eBroadcast

	logger *log.Logger
}

func NewHub() *Hub {
	return &Hub{
		clients:       make(map[*client]struct{}),
		topics:        make(map[string]*topic),
		chRegister:    make(chan *client),
		chDeregister:  make(chan *client),
		chSubscribe:   make(chan eSubscribe),
		chUnsubscribe: make(chan eUnsubscribe),
		chBroadcast:   make(chan eBroadcast),
		logger:        defaultLogger,
	}
}

func (h *Hub) SetLogger(logger *log.Logger) {
	h.logger = logger
}

func (h *Hub) Broadcast(mt int, msg []byte, topics ...string) {
	h.chBroadcast <- eBroadcast{mt, msg, topics}
}

func (h *Hub) Start() {
	for {
		select {
		case c := <-h.chRegister:
			h.register(c)

		case c := <-h.chDeregister:
			h.deregister(c)

		case e := <-h.chSubscribe:
			h.subscribe(e.client, e.topics)

		case e := <-h.chUnsubscribe:
			h.unsubscribe(e.client, e.topics)

		case e := <-h.chBroadcast:
			h.broadcast(e.mt, e.msg, e.topics)
		}
	}
}

func (h *Hub) register(c *client) {
	h.logger.Printf("wsh: register: %p", c)
	h.clients[c] = struct{}{}
}

func (h *Hub) deregister(c *client) {
	_, ok := h.clients[c]
	if !ok {
		return
	}
	h.logger.Printf("wsh: unregister: %p", c)
	delete(h.clients, c)
	close(c.send)
	for topicName := range c.topics {
		t, ok := h.topics[topicName]
		if ok {
			delete(t.clients, c)
		}
	}
}

func (h *Hub) subscribe(c *client, topics []string) {
	h.logger.Printf("wsh: subscribe: %p <- %v", c, topics)
	for _, topicName := range topics {
		c.topics[topicName] = struct{}{}
		t, ok := h.topics[topicName]
		if !ok {
			t = newTopic(topicName)
			h.topics[topicName] = t
		}
		t.clients[c] = struct{}{}
	}
}

func (h *Hub) unsubscribe(c *client, topics []string) {
	h.logger.Printf("wsh: unsubscribe: %p <- %v", c, topics)
	if len(topics) == 0 { // unsubscribe all topics
		for topicName := range c.topics {
			t, ok := h.topics[topicName]
			if ok {
				delete(t.clients, c)
			}
		}
		c.topics = make(map[string]struct{})
	} else {
		for _, topicName := range topics {
			delete(c.topics, topicName)
			t, ok := h.topics[topicName]
			if ok {
				delete(t.clients, c)
			}
		}
	}
}

func (h *Hub) broadcast(mt int, msg []byte, topics []string) {
	if len(topics) == 0 { // broadcast to all clients
		h.broadcastToClients(h.clients, mt, msg)
	} else {
		for _, topicName := range topics {
			t, ok := h.topics[topicName]
			if ok {
				h.broadcastToClients(t.clients, mt, msg)
			}
		}
	}
}

func (h *Hub) broadcastToClients(clients map[*client]struct{}, mt int, msg []byte) {
	var invalidClients []*client
	for c := range clients {
		select {
		case c.send <- message{mt, msg}:

		default:
			invalidClients = append(invalidClients, c)
		}
	}
	for _, c := range invalidClients {
		h.deregister(c)
	}
}
