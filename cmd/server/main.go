package main

import (
	"context"
	"net/http"
	"time"

	"github.com/hugjobk/wsh"
)

func main() {
	hub := wsh.NewHub()
	wsHandler := wsh.NewHandler(hub)

	go hub.Start()

	go writeToHub(hub)

	http.Handle("/ws", wsHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func writeToHub(hub *wsh.Hub) {
	t1 := time.NewTicker(1 * time.Second)
	t2 := time.NewTicker(2 * time.Second)
	t3 := time.NewTicker(4 * time.Second)
	for {
		select {
		case <-t1.C:
			hub.Broadcast(context.Background(), wsh.TextMessage, []byte("Topic A"), "A")
		case <-t2.C:
			hub.Broadcast(context.Background(), wsh.TextMessage, []byte("Topic B"), "B")
		case <-t3.C:
			hub.Broadcast(context.Background(), wsh.TextMessage, []byte("All topics"))
		}
	}
}
