package handlers

import (
	"strconv"
	"time"

	"github.com/gorilla/schema"
	"github.com/vancomm/minesweeper-server/internal/mines"
)

type CreateNewGameDTO struct {
	Width     int  `schema:"width,required"`
	Height    int  `schema:"height,required"`
	MineCount int  `schema:"mine_count,required"`
	Unique    bool `schema:"unique,required"`
}

func ParseCreateNewGameDTO(src map[string][]string) (CreateNewGameDTO, error) {
	var dto CreateNewGameDTO
	dec := schema.NewDecoder()
	dec.IgnoreUnknownKeys(true)
	err := dec.Decode(&dto, src)
	return dto, err
}

type GameSessionDTO struct {
	GameSessionId string         `json:"game_session_id"`
	Grid          mines.GridInfo `json:"grid"`
	Width         int            `json:"width"`
	Height        int            `json:"height"`
	MineCount     int            `json:"mine_count"`
	Unique        bool           `json:"unique"`
	Dead          bool           `json:"dead"`
	Won           bool           `json:"won"`
	StartedAt     int64          `json:"started_at"`
	EndedAt       *int64         `json:"ended_at,omitempty"`
}

func NewGameSessionDTO(
	gameSessionID int64,
	startedAt time.Time,
	endedAt *time.Time,
	g *mines.GameState,
) *GameSessionDTO {
	var endedAtInt *int64
	if endedAt != nil {
		e := endedAt.UnixMilli()
		endedAtInt = &e
	}
	dto := &GameSessionDTO{
		GameSessionId: strconv.FormatInt(gameSessionID, 10),
		StartedAt:     startedAt.UnixMilli(),
		EndedAt:       endedAtInt,
		Grid:          g.PlayerGrid,
		Width:         g.Width,
		Height:        g.Height,
		MineCount:     g.MineCount,
		Unique:        g.Unique,
		Dead:          g.Dead,
		Won:           g.Won,
	}
	return dto
}
