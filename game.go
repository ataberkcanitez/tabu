package main

import (
	"encoding/json"
	"math/rand"
	"time"
)

type ClientList map[*Client]bool

func (cl ClientList) MarshalJSON() ([]byte, error) {
	var clients []string
	for client := range cl {
		clients = append(clients, client.Username)
	}
	return json.Marshal(clients)
}

type Game struct {
	GameId     int        `json:"game_id"`
	RedTeam    ClientList `json:"red_team"`
	BlueTeam   ClientList `json:"blue_team"`
	AllPlayers ClientList `json:"all_players"`
	IsStarted  bool       `json:"is_started"`
}

func NewGame() *Game {
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	gameId := rng.Intn(999) + 10000
	return &Game{
		GameId:     gameId,
		RedTeam:    make(ClientList),
		BlueTeam:   make(ClientList),
		AllPlayers: make(ClientList),
		IsStarted:  false,
	}
}

func (g *Game) SwitchTeam(player *Client, selectedTeam string) (*Game, error) {
	delete(g.RedTeam, player)
	delete(g.BlueTeam, player)
	if selectedTeam == "red" {
		g.RedTeam[player] = true
	} else {
		g.BlueTeam[player] = true
	}

	return g, nil
}
