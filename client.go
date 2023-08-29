package wsh

import (
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type message struct {
	mt   int
	data []byte
}

type client struct {
	hub    *Hub
	ws     *websocket.Conn
	topics map[string]struct{}
	send   chan message

	logger *log.Logger

	readLimit  int64
	writeWait  time.Duration
	pongWait   time.Duration
	pingPeriod time.Duration
}

func (c *client) readPump() {
	defer func() {
		c.hub.chDeregister <- c
		c.ws.Close()
	}()

	c.ws.SetReadLimit(c.readLimit)
	c.ws.SetReadDeadline(time.Now().Add(c.pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(c.pongWait))
		return nil
	})

	for {
		mt, msg, err := c.ws.ReadMessage()
		if err != nil {
			c.logger.Printf("wsh: read failed: %s", err)
			break
		}

		switch mt {
		case PingMessage:
			c.ws.WriteMessage(PongMessage, nil)

		case TextMessage:
			if len(msg) == 0 {
				continue
			}
			cmd, args := parseCmd(msg)
			switch strings.ToLower(cmd) {
			case "subscribe":
				c.hub.chSubscribe <- eSubscribe{c, args}

			case "unsubscribe":
				c.hub.chUnsubscribe <- eUnsubscribe{c, args}

			default:
				c.logger.Printf("wsh: unhandled message: %s", string(msg))
			}
		}
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(c.pingPeriod)

	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.write(CloseMessage, nil)
				return
			}
			if err := c.write(msg.mt, msg.data); err != nil {
				c.logger.Printf("wsh: write failed: %s", err)
				return
			}

		case <-ticker.C:
			if err := c.write(PingMessage, nil); err != nil {
				c.logger.Printf("wsh: ping failed: %s", err)
				return
			}
		}
	}
}

func (c *client) write(mt int, message []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(c.writeWait))
	return c.ws.WriteMessage(mt, message)
}

func parseCmd(data []byte) (cmd string, args []string) {
	s := strings.Split(string(data), " ")
	return s[0], s[1:]
}
