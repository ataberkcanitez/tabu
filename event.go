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
	EventGameCanStart    = "game_can_start"
	EventRoundEnd        = "round_end"
	EventCorrect         = "correct"
	EventIncorrect       = "incorrect"
	EventPass            = "pass"
	EventScoreUpdate     = "score_update"
	EventRoundUpdate     = "round"
	EventStartNewRound   = "start_new_round"
)

type SelectTeamEvent struct {
	Team string `json:"team"`
}

type RoundEndEvent struct {
}
