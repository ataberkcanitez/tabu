package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

var (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10
)

type Client struct {
	connection *websocket.Conn
	manager    *Manager
	gameId     int
	Username   string
	Team       string

	// egress is used to avoid concurrent writes on the websocket connection
	egress chan Event
}

func NewClient(conn *websocket.Conn, manager *Manager, gameId int, username string) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan Event),
		gameId:     gameId,
		Username:   username,
		Team:       "not_selected",
	}
}

func (c *Client) ReadEvents() {
	defer func() {
		//c.manager.removeClient(c)
	}()

	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println("failed to set read deadline: ", err)
		return
	}

	c.connection.SetReadLimit(512)
	c.connection.SetPongHandler(c.pongHandler)

	for {
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("err reading message: %v\n", err)
			}
			break
		}

		var request Event
		if err := json.Unmarshal(payload, &request); err != nil {
			log.Printf("failed to unmarshall payload: %v\n", err)
			break
		}

		if err := c.manager.routeEvent(request, c); err != nil {
			log.Printf("failed to route event: %v\n", err)
		}
	}
}

func (c *Client) WriteEvents() {
	defer func() {
		//c.manager.removeClient(c)
	}()

	ticker := time.NewTicker(pingInterval)
	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("failed to write close message: ", err)
				}
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("failed to marshal payload: %v\n", err)
				return
			}
			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("failed to write message: ", err)
			}
		case <-ticker.C:
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println("failed to send ping: ", err)
				return
			}
		}
	}
}

func (c *Client) pongHandler(string) error {
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
