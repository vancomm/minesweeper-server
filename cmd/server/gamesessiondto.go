package main

import (
	"bytes"
	"encoding/gob"
	"strconv"

	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

type gameSessionDTO struct {
	GameSessionId string     `json:"game_session_id"`
	Grid          mines.Grid `json:"grid"`
	Width         int        `json:"width"`
	Height        int        `json:"height"`
	MineCount     int        `json:"mine_count"`
	Unique        bool       `json:"unique"`
	Dead          bool       `json:"dead"`
	Won           bool       `json:"won"`
	StartedAt     int64      `json:"started_at"`
	EndedAt       *int64     `json:"ended_at,omitempty"`
}

func NewGameSessionDTO(s repository.GameSession) (*gameSessionDTO, error) {
	var state mines.GameState
	buf := bytes.NewBuffer(s.State)
	if err := gob.NewDecoder(buf).Decode(&state); err != nil {
		return nil, err
	}

	var endedAt *int64
	if !s.EndedAt.Time.IsZero() {
		e := s.EndedAt.Time.UnixMilli()
		endedAt = &e
	}

	dto := &gameSessionDTO{
		GameSessionId: strconv.Itoa(s.GameSessionId),
		Grid:          state.PlayerGrid,
		Width:         s.Width,
		Height:        s.Height,
		MineCount:     s.MineCount,
		Unique:        s.Unique,
		Dead:          s.Dead,
		Won:           s.Won,
		StartedAt:     s.StartedAt.Time.UnixMilli(),
		EndedAt:       endedAt,
	}
	return dto, nil
}
