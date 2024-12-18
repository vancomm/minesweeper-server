package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/vancomm/minesweeper-server/internal/mines"
)

type GameSession struct {
	SessionId int
	PlayerId  *int
	State     mines.GameState
	StartedAt time.Time
	EndedAt   time.Time
}

type GameSessionJSON struct {
	SessionId string         `json:"session_id"`
	Grid      mines.GridInfo `json:"grid"`
	Width     int            `json:"width"`
	Height    int            `json:"height"`
	MineCount int            `json:"mine_count"`
	Unique    bool           `json:"unique"`
	Dead      bool           `json:"dead"`
	Won       bool           `json:"won"`
	StartedAt int64          `json:"started_at"`
	EndedAt   *int64         `json:"ended_at,omitempty"`
}

func (s GameSession) MarshalJSON() ([]byte, error) {
	var endedAt *int64
	if !s.EndedAt.IsZero() {
		e := s.EndedAt.UnixMilli()
		endedAt = &e
	}
	return json.Marshal(GameSessionJSON{
		SessionId: strconv.Itoa(s.SessionId),
		Grid:      s.State.PlayerGrid,
		Width:     s.State.Width,
		Height:    s.State.Height,
		MineCount: s.State.MineCount,
		Unique:    s.State.Unique,
		Dead:      s.State.Dead,
		Won:       s.State.Won,
		StartedAt: s.StartedAt.UnixMilli(),
		EndedAt:   endedAt,
	})
}
