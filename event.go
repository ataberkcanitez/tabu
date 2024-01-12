package main

import "encoding/json"

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(event Event, p *Player) error

const (
	EventSelectTeam      = "select_team"
	EventTeamUpdate      = "team_update"
	EventReady           = "ready"
	EventGameStart       = "game_start"
	EventGameStartUpdate = "game_start_update"
	EventRoundEnd        = "round_end"
	EventCorrect         = "correct"
	EventIncorrect       = "incorrect"
	EventScoreUpdate     = "score_update"
)

type SelectTeamEvent struct {
	Team string `json:"team"`
}

type GameStartEvent struct {
}

type RoundEndEvent struct {
}
