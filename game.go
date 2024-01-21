package main

import (
	"encoding/json"
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
	Narrator   string         `json:"narrator"`
	Round      *Round         `json:"round"`
	Timer      *time.Timer    `json:"-"`
	RoundIdx   int            `json:"-"`
}

type Round struct {
	Taboo        Taboo  `json:"taboo"`
	RedTeamTurn  bool   `json:"red_team_turn"`
	BlueTeamTurn bool   `json:"blue_team_turn"`
	Narrator     string `json:"narrator"`
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
		Round:      nil,
		Timer:      nil,
		RoundIdx:   0,
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
	if len(g.AllPlayers) < 2 {
		return false
	}

	if len(g.RedTeam) < 1 || len(g.BlueTeam) < 1 {
		return false
	}

	okToStart := true
	for player := range g.AllPlayers {
		if !player.ready {
			okToStart = false
			break
		}
	}
	return okToStart
}

func (g *Game) Start() {
	defer func() {
		g.endGame()
	}()

	g.startRoundForRedTeam()
}

func (g *Game) endGame() {
	for player := range g.AllPlayers {
		player.ready = false
		player.Team = "not_selected"
	}
	g.IsStarted = false
}

func (g *Game) IncreaseScore() error {
	team := ""
	if g.Round.RedTeamTurn {
		team = "red"
	} else {
		team = "blue"
	}
	g.Scores[team] += 1
	taboo := g.selectRandomTaboo(TaboosFromJson())
	g.Round.Taboo = taboo

	return nil
}

func (g *Game) DecreaseScore() error {
	team := ""
	if g.Round.RedTeamTurn {
		team = "red"
	} else {
		team = "blue"
	}

	g.Scores[team] -= 1

	taboo := g.selectRandomTaboo(TaboosFromJson())
	g.Round.Taboo = taboo
	return nil
}

func (g *Game) Pass() error {
	taboo := g.selectRandomTaboo(TaboosFromJson())
	g.Round.Taboo = taboo
	return nil
}

func (g *Game) startRoundForRedTeam() {
	taboos := TaboosFromJson()
	currentTaboo := g.selectRandomTaboo(taboos)
	redTeamIdx := g.RoundIdx / 2
	round := &Round{
		Taboo:        currentTaboo,
		RedTeamTurn:  true,
		BlueTeamTurn: false,
		Narrator:     "",
	}
	g.Round = round
	for player, _ := range g.RedTeam {
		if g.RoundIdx == redTeamIdx {
			g.Narrator = player.Username
			round.Narrator = player.Username
			break
		}
	}

	g.RoundIdx++

	g.NotifyPlayersForRoundEvent()

	g.Timer = time.NewTimer(10 * time.Second)
	go func() {
		<-g.Timer.C
		g.ChangeTeamAndStartNewRound()
	}()
}

func (g *Game) startRoundForBlueTeam() {
	taboos := TaboosFromJson()
	currentTaboo := g.selectRandomTaboo(taboos)
	blueTeamIdx := (g.RoundIdx - 1) / 2
	round := &Round{
		Taboo:        currentTaboo,
		RedTeamTurn:  false,
		BlueTeamTurn: true,
		Narrator:     "",
	}

	g.Round = round
	for player, _ := range g.BlueTeam {
		if g.RoundIdx == blueTeamIdx {
			g.Narrator = player.Username
			round.Narrator = player.Username
			break
		}
	}

	g.RoundIdx++
	g.NotifyPlayersForRoundEvent()

	g.Timer = time.NewTimer(10 * time.Second)
	go func() {
		<-g.Timer.C
		g.ChangeTeamAndStartNewRound()
	}()
}

func (g *Game) NotifyPlayersForRoundEvent() {
	data, err := json.Marshal(g)
	if err != nil {
		fmt.Printf("failed to marshal payload: %v\n", err)
		return
	}
	for player, _ := range g.AllPlayers {
		player.egress <- Event{
			Type:    EventRoundUpdate,
			Payload: data,
		}
	}
}

func (g *Game) selectRandomTaboo(taboos []Taboo) Taboo {
	s := rand.NewSource(time.Now().Unix())
	r := rand.New(s)
	return taboos[r.Intn(len(taboos))]
}

func (g *Game) ChangeTeamAndStartNewRound() {
	if g.Timer != nil {
		g.Timer.Stop()
	}

	g.NotifyPlayersRoundFinished()
}

func (g *Game) NotifyPlayersRoundFinished() {
	type RoundFinished struct {
		Round         int `json:"round"`
		RedTeamScore  int `json:"red_team_score"`
		BlueTeamScore int `json:"blue_team_score"`
	}

	roundFinished := RoundFinished{
		Round:         g.RoundIdx,
		RedTeamScore:  g.Scores["red"],
		BlueTeamScore: g.Scores["blue"],
	}
	data, err := json.Marshal(roundFinished)
	if err != nil {
		fmt.Printf("failed to marshal payload: %v\n", err)
		return
	}
	for player, _ := range g.AllPlayers {
		player.egress <- Event{
			Type:    EventRoundEnd,
			Payload: data,
		}
	}
}

// TODO: change timer from 10 seconds to 60 seconds.
// TODO: every time round changes, we are not able to update the narrator. update the narrator.
// check users permissions to see the taboo card, now it is broken.
