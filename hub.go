package wsh

import (
	"context"
	"log"
)

type eBroadcast struct {
	ctx    context.Context
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
	chResponse    chan struct{} // Response after broastcasting a message to clients

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
		chResponse:    make(chan struct{}),
		logger:        defaultLogger,
	}
}

func (h *Hub) SetLogger(logger *log.Logger) {
	h.logger = logger
}

func (h *Hub) Broadcast(ctx context.Context, mt int, msg []byte, topics ...string) {
	h.chBroadcast <- eBroadcast{ctx, mt, msg, topics}
	<-h.chResponse
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
			h.broadcast(e.ctx, e.mt, e.msg, e.topics)
			h.chResponse <- struct{}{}
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
	h.logger.Printf("wsh: deregister: %p", c)
	delete(h.clients, c)
	close(c.chSend)
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

func (h *Hub) broadcast(ctx context.Context, mt int, msg []byte, topics []string) {
	if len(topics) == 0 { // broadcast to all clients
		h.broadcastToClients(ctx, h.clients, mt, msg)
	} else {
		for _, topicName := range topics {
			t, ok := h.topics[topicName]
			if ok {
				h.broadcastToClients(ctx, t.clients, mt, msg)
			}
		}
	}
}

func (h *Hub) broadcastToClients(ctx context.Context, clients map[*client]struct{}, mt int, msg []byte) {
	var invalidClients []*client
	for c := range clients {
		c.chSend <- message{mt, msg}
	}
	for c := range clients {
		select {
		case _, ok := <-c.chResponse:
			if !ok {
				invalidClients = append(invalidClients, c)
			}
		default:
			select {
			case _, ok := <-c.chResponse:
				if !ok {
					invalidClients = append(invalidClients, c)
				}
			case <-ctx.Done():
				invalidClients = append(invalidClients, c)
			}
		}
	}
	for _, c := range invalidClients {
		h.deregister(c)
	}
}
