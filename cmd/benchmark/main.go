package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hugjobk/wsh"
)

const (
	MSG_SIZE     = 100
	TOPIC_COUNT  = 100
	CLIENT_COUNT = 10000
)

var msg = make([]byte, MSG_SIZE)

var counter uint64

func main() {
	hub := wsh.NewHub()
	wsHandler := wsh.NewHandler(hub)

	go hub.Start()

	go monitor()

	go startBenchmark(hub)

	http.Handle("/ws", wsHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func monitor() {
	var curr, prev uint64
	t := time.NewTicker(1 * time.Second)
	for range t.C {
		curr = atomic.LoadUint64(&counter)
		fmt.Printf("counter=%d delta=%d\n", curr, curr-prev)
		prev = curr
	}
}

func startBenchmark(hub *wsh.Hub) {
	connectToServer()
	writeToHub(hub)
}

func connectToServer() {
	for i := 0; i < CLIENT_COUNT; i++ {
		conn, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:8080/ws", nil)
		if err != nil {
			log.Print(err)
			continue
		}
		go func() {
			defer conn.Close()
			cmd := fmt.Sprintf("subscribe %s", randomTopic())
			if err := conn.WriteMessage(wsh.TextMessage, []byte(cmd)); err != nil {
				log.Print(err)
				return
			}
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					log.Print(err)
					return
				}
				atomic.AddUint64(&counter, 1)
			}
		}()
	}
}

func writeToHub(hub *wsh.Hub) {
	for {
		hub.Broadcast(context.Background(), wsh.TextMessage, msg, randomTopic())
	}
}

func randomTopic() string {
	return fmt.Sprintf("topic-%d", rand.Intn(TOPIC_COUNT))
}
