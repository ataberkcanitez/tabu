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

type Player struct {
	connection *websocket.Conn
	manager    *Manager
	gameId     int
	Username   string
	Team       string
	egress     chan Event
	ready      bool
}

type PlayerList map[*Player]bool

func NewPlayer(conn *websocket.Conn, manager *Manager, gameId int, username string) *Player {
	return &Player{
		connection: conn,
		manager:    manager,
		egress:     make(chan Event),
		gameId:     gameId,
		Username:   username,
		Team:       "not_selected",
		ready:      false,
	}
}

func (p *Player) ReadEvents() {
	defer func() {
		p.manager.removePlayer(p)
	}()

	if err := p.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println("failed to set read deadline: ", err)
		return
	}

	p.connection.SetReadLimit(512)
	p.connection.SetPongHandler(p.pongHandler)

	for {
		_, payload, err := p.connection.ReadMessage()
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

		if err := p.manager.routeEvent(request, p); err != nil {
			log.Printf("failed to route event: %v\n", err)
		}
	}
}

func (p *Player) WriteEvents() {
	defer func() {
		p.manager.removePlayer(p)
	}()

	ticker := time.NewTicker(pingInterval)
	for {
		select {
		case message, ok := <-p.egress:
			if !ok {
				if err := p.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("failed to write close message: ", err)
				}
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("failed to marshal payload: %v\n", err)
				return
			}
			if err := p.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("failed to write message: ", err)
			}
		case <-ticker.C:
			if err := p.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println("failed to send ping: ", err)
				return
			}
		}
	}
}

func (p *Player) pongHandler(string) error {
	return p.connection.SetReadDeadline(time.Now().Add(pongWait))
}

func (pl PlayerList) MarshalJSON() ([]byte, error) {
	type PlayerDetails struct {
		Username string `json:"username"`
		Ready    bool   `json:"ready"`
	}
	var players []PlayerDetails
	for player := range pl {
		players = append(players, PlayerDetails{Username: player.Username, Ready: player.ready})
	}
	return json.Marshal(players)
}
