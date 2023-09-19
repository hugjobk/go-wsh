package wsh

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Handler struct {
	hub *Hub

	logger   *log.Logger
	upgrader websocket.Upgrader

	readLimit  int64
	writeWait  time.Duration
	pongWait   time.Duration
	pingPeriod time.Duration
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub:        hub,
		logger:     defaultLogger,
		upgrader:   defaultUpgrader,
		readLimit:  defaultReadLimit,
		writeWait:  defaultWriteWait,
		pongWait:   defaultPongWait,
		pingPeriod: defaultPingPeriod,
	}
}

func (h *Handler) SetLogger(logger *log.Logger) {
	h.logger = logger
}

func (h *Handler) SetUpgrader(upgrader websocket.Upgrader) {
	h.upgrader = upgrader
}

func (h *Handler) SetReadLimit(readLimit int64) {
	h.readLimit = readLimit
}

func (h *Handler) SetWriteWait(writeWait time.Duration) {
	h.writeWait = writeWait
}

func (h *Handler) SetPongWait(pongWait time.Duration) {
	h.pongWait = pongWait
}

func (h *Handler) SetPingPeriod(pingPeriod time.Duration) {
	h.pingPeriod = pingPeriod
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Printf("wsh: upgrade failed: %s", err)
		return
	}

	c := &client{
		hub:        h.hub,
		ws:         ws,
		topics:     make(map[string]struct{}),
		chSend:     make(chan message, 1),
		chResponse: make(chan struct{}),
		logger:     h.logger,
		readLimit:  h.readLimit,
		writeWait:  h.writeWait,
		pongWait:   h.pongWait,
		pingPeriod: h.pingPeriod,
	}

	h.hub.chRegister <- c

	go c.writePump()
	c.readPump()
}
