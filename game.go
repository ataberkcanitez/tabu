package main

import (
	"fmt"
	"math/rand"
	"time"
)

type Game struct {
	GameId     int            `json:"game_id"`
	RedTeam    PlayerList     `json:"red_team"`
	BlueTeam   PlayerList     `json:"blue_team"`
	AllPlayers PlayerList     `json:"all_players"`
	IsStarted  bool           `json:"is_started"`
	Scores     map[string]int `json:"scores"`
}

func NewGame() *Game {
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	gameId := rng.Intn(999) + 10000
	scoresMap := make(map[string]int)
	scoresMap["red"] = 0
	scoresMap["blue"] = 0
	return &Game{
		GameId:     gameId,
		RedTeam:    make(PlayerList),
		BlueTeam:   make(PlayerList),
		AllPlayers: make(PlayerList),
		IsStarted:  false,
		Scores:     scoresMap,
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

func (g *Game) CanGameStart() bool {
	okToStart := true
	for player := range g.AllPlayers {
		if !player.ready {
			okToStart = false
			break
		}
	}
	g.IsStarted = okToStart
	return okToStart
}

func (g *Game) Start() {
	defer func() {
		g.endGame()
	}()

	for {

	}
}

func (g *Game) endGame() {
	for player := range g.AllPlayers {
		player.ready = false
		player.Team = "not_selected"
		player.narrator = false
	}
	g.IsStarted = false
}

func (g *Game) IncreaseScore(team string) error {
	if !g.IsStarted {
		return fmt.Errorf("game is not started")
	}
	g.Scores[team] += 1
	return nil
}

func (g *Game) DecreaseScore(team string) error {
	if !g.IsStarted {
		return fmt.Errorf("game is not started")
	}
	g.Scores[team] -= 1
	return nil
}
