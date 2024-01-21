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
	m.handlers[EventReady] = Ready
	m.handlers[EventGameStart] = GameStart
	m.handlers[EventCorrect] = CorrectGuess
	m.handlers[EventIncorrect] = IncorrectGuess
	m.handlers[EventPass] = PassTheCard
	m.handlers[EventStartNewRound] = StartNewRound
}

func StartNewRound(event Event, p *Player) error {
	game := p.manager.Games[p.gameId]
	if game.Round.RedTeamTurn {
		game.startRoundForBlueTeam()
	} else {
		game.startRoundForRedTeam()
	}
	return nil
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
	log.Printf("New connection: username: %s\n", username)
	conn, err := websocketUpgrade.Upgrade(writer, request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	player := NewPlayer(conn, m, gameIdInt, username)
	m.addPlayer(gameIdInt, player)

	game := m.Games[gameIdInt]
	data, err := json.Marshal(game)
	if err != nil {
		log.Println(err)
		return
	}

	go player.ReadEvents()
	go player.WriteEvents()

	sendEventToSinglePlayer(EventTeamUpdate, data, player)
}

func (m *Manager) addPlayer(gameId int, player *Player) {
	m.Lock()
	defer m.Unlock()
	game := m.Games[gameId]
	game.AllPlayers[player] = true
}

func (m *Manager) routeEvent(event Event, p *Player) interface{} {
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, p); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("there is no such event type : %v", event.Type)
}

func (m *Manager) removePlayer(player *Player) {
	m.Lock()
	defer m.Unlock()
	for _, game := range m.Games {
		if _, ok := game.RedTeam[player]; ok {
			delete(game.RedTeam, player)
		}
		if _, ok := game.BlueTeam[player]; ok {
			delete(game.BlueTeam, player)
		}

		if _, ok := game.AllPlayers[player]; ok {
			delete(game.AllPlayers, player)
		}
	}

	game := m.Games[player.gameId]
	data, err := json.Marshal(game)
	if err != nil {
		log.Println(err)
		return
	}
	sendEvent(EventTeamUpdate, data, game.AllPlayers)
}

func checkOrigin(_ *http.Request) bool {
	return true
}

//--- Events

func SelectTeam(event Event, player *Player) error {
	var selectTeamEvent SelectTeamEvent
	if err := json.Unmarshal(event.Payload, &selectTeamEvent); err != nil {
		return err
	}

	team := selectTeamEvent.Team
	if team == "" || team != "red" && team != "blue" {
		return fmt.Errorf("invalid team")
	}
	player.Team = team
	gameId := player.gameId
	game := player.manager.Games[gameId]
	switchedTeam, err := game.SwitchTeam(player, team)
	if err != nil {
		return err
	}
	data, err := json.Marshal(switchedTeam)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	sendEvent(EventTeamUpdate, data, game.AllPlayers)
	//
	canGameStart(game)
	return nil
}

func Ready(_ Event, p *Player) error {
	if p.Team == "" || p.Team == "not_selected" {
		sendEventToSinglePlayer("ReadyUpdateError", []byte(`{"error": "player has not selected a team"}`), p)
		return fmt.Errorf("player has not selected a team")
	}
	p.ready = true
	game := p.manager.Games[p.gameId]
	data, err := json.Marshal(game)
	if err != nil {
		log.Println(err)
		return err
	}

	sendEvent(EventTeamUpdate, data, game.AllPlayers)
	canGameStart(game)

	return nil
}

func canGameStart(game *Game) error {
	type GameCanStart struct {
		CanStart bool `json:"can_start"`
	}

	canGameStart := &GameCanStart{
		CanStart: game.CanGameStart(),
	}
	data, err := json.Marshal(canGameStart)
	if err != nil {
		log.Println(err)
		return err
	}
	sendEvent(EventGameCanStart, data, game.AllPlayers)
	return nil
}

func GameStart(_ Event, p *Player) error {
	game := p.manager.Games[p.gameId]
	canGameStart := game.CanGameStart()
	if !canGameStart {
		return fmt.Errorf("not all players are ready")
	}

	game.IsStarted = true
	go game.Start()

	data, err := json.Marshal(game)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}
	sendEvent(EventGameStartUpdate, data, game.AllPlayers)
	return nil
}

func CorrectGuess(_ Event, p *Player) error {
	game := p.manager.Games[p.gameId]
	err := game.IncreaseScore()
	if err != nil {
		log.Println(err)
		return err
	}

	data, err := prepareScoreUpdateResponse(game)
	if err != nil {
		log.Println(err)
		return err
	}

	sendEvent(EventScoreUpdate, data, game.AllPlayers)
	return nil
}

func IncorrectGuess(_ Event, p *Player) error {
	game := p.manager.Games[p.gameId]
	err := game.DecreaseScore()
	if err != nil {
		log.Println(err)
		return err
	}

	data, err := prepareScoreUpdateResponse(game)
	if err != nil {
		log.Println(err)
		return err
	}

	sendEvent(EventScoreUpdate, data, game.AllPlayers)
	return nil
}

func PassTheCard(event Event, p *Player) error {
	game := p.manager.Games[p.gameId]
	err := game.Pass()
	if err != nil {
		log.Println(err)
		return err
	}

	data, err := prepareScoreUpdateResponse(game)
	if err != nil {
		log.Println(err)
		return err
	}

	sendEvent(EventScoreUpdate, data, game.AllPlayers)
	return nil
}

func prepareScoreUpdateResponse(game *Game) (json.RawMessage, error) {
	type ScoreUpdate struct {
		Scores map[string]int `json:"scores"`
		Taboo  Taboo          `json:"taboo"`
	}

	resp := &ScoreUpdate{
		Scores: game.Scores,
		Taboo:  game.Round.Taboo,
	}

	return json.Marshal(resp)
}

func sendEvent(eventType string, payload json.RawMessage, players PlayerList) {
	outgoingEvent := Event{
		Type:    eventType,
		Payload: payload,
	}
	for p := range players {
		p.egress <- outgoingEvent
	}
}

func sendEventToSinglePlayer(eventType string, payload json.RawMessage, player *Player) {
	outgoingEvent := Event{
		Type:    eventType,
		Payload: payload,
	}
	player.egress <- outgoingEvent
}
