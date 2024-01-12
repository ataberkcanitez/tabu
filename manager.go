package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"sync"
)

var (
	websocketUpgrade = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	sync.RWMutex
	handlers map[string]EventHandler
	Games    map[int]*Game
}

func NewManager() *Manager {
	m := &Manager{
		handlers: make(map[string]EventHandler),
		Games:    make(map[int]*Game),
	}
	m.setupEventHandlers()
	return m
}

func (m *Manager) setupEventHandlers() {
	m.handlers[EventSelectTeam] = SelectTeam
	//m.handlers[Event]
}

func (m *Manager) CreateGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	game := NewGame()
	m.Games[game.GameId] = game
	type response struct {
		Game Game `json:"game"`
	}

	resp := &response{
		Game: *game,
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (m *Manager) serveWS(writer http.ResponseWriter, request *http.Request) {
	gameId := request.URL.Query().Get("game_id")
	if gameId == "" {
		game := NewGame()
		gameId = strconv.Itoa(game.GameId)
		return
	}

	gameIdInt, err := strconv.Atoi(gameId)
	if err != nil {
		log.Println(err)
		return
	}

	username := request.URL.Query().Get("username")
	if username == "" {
		http.Error(writer, "Missing username", http.StatusBadRequest)
		return
	}
	log.Println("New Connection")
	conn, err := websocketUpgrade.Upgrade(writer, request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, m, gameIdInt, username)
	m.addClient(gameIdInt, client)

	go client.ReadEvents()
	go client.WriteEvents()

}

func (m *Manager) addClient(gameId int, client *Client) {
	m.Lock()
	defer m.Unlock()
	game := m.Games[gameId]
	game.AllPlayers[client] = true
}

func (m *Manager) routeEvent(event Event, c *Client) interface{} {
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("there is no such event type")
}

func (m *Manager) removeClient(c *Client) {
	m.Lock()
	defer m.Unlock()
	for _, game := range m.Games {
		if _, ok := game.RedTeam[c]; ok {
			delete(game.RedTeam, c)
		}
		if _, ok := game.BlueTeam[c]; ok {
			delete(game.BlueTeam, c)
		}

		if _, ok := game.AllPlayers[c]; ok {
			delete(game.AllPlayers, c)
		}
	}

}

func checkOrigin(_ *http.Request) bool {
	return true
}

//--- Events

func SelectTeam(event Event, client *Client) error {
	var selectTeamEvent SelectTeamEvent
	if err := json.Unmarshal(event.Payload, &selectTeamEvent); err != nil {
		return err
	}

	team := selectTeamEvent.Team
	if team == "" || team != "red" && team != "blue" {
		return fmt.Errorf("invalid team")
	}
	client.Team = team
	gameId := client.gameId
	game := client.manager.Games[gameId]
	switchedTeam, err := game.SwitchTeam(client, team)
	if err != nil {
		return err
	}
	data, err := json.Marshal(switchedTeam)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	outgoingEvent := Event{
		Type:    EventTeamUpdate,
		Payload: data,
	}

	for client := range game.AllPlayers {
		client.egress <- outgoingEvent
	}

	return nil
}
