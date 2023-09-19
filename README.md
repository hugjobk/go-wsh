# Websocket handler

## Implementation of golang websocket server that allows clients to subscribe/unsubscribe to multiple topics

## Start websocket server

```
go run cmd/server/main.go
```

## Connect to websocket server

```
wscat -c 127.0.0.1:8080/ws
```

## Subscribe to one or more topics

```
subscribe A B
```

## Unsubscribe from one or more topics

```
unsubscribe B
```

## Unsubscribe from all topics
```
unsubscribe
```
