package main

import (
	"encoding/json"
	"math/rand"
	"time"
)

type PlayerList map[*Player]bool

func (pl PlayerList) MarshalJSON() ([]byte, error) {
	var players []string
	for player := range pl {
		players = append(players, player.Username)
	}
	return json.Marshal(players)
}

type Game struct {
	GameId     int        `json:"game_id"`
	RedTeam    PlayerList `json:"red_team"`
	BlueTeam   PlayerList `json:"blue_team"`
	AllPlayers PlayerList `json:"all_players"`
	IsStarted  bool       `json:"is_started"`
}

func NewGame() *Game {
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	gameId := rng.Intn(999) + 10000
	return &Game{
		GameId:     gameId,
		RedTeam:    make(PlayerList),
		BlueTeam:   make(PlayerList),
		AllPlayers: make(PlayerList),
		IsStarted:  false,
	}
}

func (g *Game) SwitchTeam(player *Player, selectedTeam string) (*Game, error) {
	delete(g.RedTeam, player)
	delete(g.BlueTeam, player)
	if selectedTeam == "red" {
		g.RedTeam[player] = true
	} else {
		g.BlueTeam[player] = true
	}

	return g, nil
}
