package main

import (
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/vancomm/minesweeper-server/internal/mines"
	"github.com/vancomm/minesweeper-server/internal/repository"
)

func iterBySep(s string, sep string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		i := 0
		found := true
		var piece string
		for found {
			piece, s, found = strings.Cut(s, sep)
			if !yield(i, piece) {
				return
			}
			i += 1
		}
	}
}

func parseXY(twoStrings []string) (x int, y int, err error) {
	if x, err = strconv.Atoi(twoStrings[0]); err != nil {
		err = fmt.Errorf("first argument must be an int")
		return
	}
	if y, err = strconv.Atoi(twoStrings[1]); err != nil {
		err = fmt.Errorf("second argument must be an int")
		return
	}
	return
}

var commandNargs = map[string]int{
	"g": 0,
	"o": 2,
	"f": 2,
	"c": 2,
	"r": 0,
}

func parseCommand(g *mines.GameState, c string) error {
	parts := strings.Split(c, " ")

	nargs, ok := commandNargs[parts[0]]
	if !ok {
		return fmt.Errorf("unknown command")
	}
	if nargs != len(parts)-1 {
		return fmt.Errorf("invalid number of arguments")
	}

	switch parts[0] {
	case "g":
		return nil
	case "o":
		x, y, err := parseXY(parts[1:])
		if err != nil {
			return err
		}
		if !g.ValidatePosition(x, y) {
			return fmt.Errorf("invalid square coordinates")
		}
		g.OpenCell(x, y)
		return nil
	case "f":
		if x, y, err := parseXY(parts[1:]); err != nil {
			return err
		} else if !g.ValidatePosition(x, y) {
			return fmt.Errorf("invalid square coordinates")
		} else {
			g.FlagCell(x, y)
		}
		return nil
	case "c":
		if x, y, err := parseXY(parts[1:]); err != nil {
			return err
		} else if !g.ValidatePosition(x, y) {
			return fmt.Errorf("invalid square coordinates")
		} else {
			g.ChordCell(x, y)
		}
		return nil
	case "r":
		g.RevealMines()
		return nil
	}
	return fmt.Errorf("invalid command")
}

func (g GameHandler) ConnectWS(w http.ResponseWriter, r *http.Request) {
	sessionId, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	session, err := g.repo.GetSession(r.Context(), sessionId)
	if err == pgx.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		g.logger.Error("could not fetch session from db", slog.Any("error", err))
		return
	}

	game, err := mines.ParseGameStateFromBytes(session.State)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	c, err := g.ws.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		g.logger.Error("unable to upgrade", slog.Any("error", err))
		return
	}

	defer c.Close()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				g.logger.Warn("abnormal ws break", slog.Any("error", err))
			}
			break
		}
		if mt != websocket.TextMessage {
			break
		}
		text := strings.TrimSpace(string(message))
		g.logger.Debug(fmt.Sprintf("\t> %s", text))
		for _, c := range iterBySep(text, "\n") {
			if err := parseCommand(game, c); err != nil {
				g.logger.Error("unable to process command", slog.Any("error", err))
				return
			}
			if game.Won || game.Dead {
				*session.EndedAt = time.Now().UTC()
				game.RevealMines()
				break
			}
		}

		b, err := game.Bytes()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			g.logger.Error("unable to serialize game state", slog.Any("error", err))
			return
		}

		err = g.repo.UpdateSession(r.Context(), repository.UpdateSessionParams{
			GameSessionID: session.GameSessionID,
			State:         b,
			Dead:          game.Dead,
			Won:           game.Won,
			EndedAt:       session.EndedAt,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			g.logger.Error("unable to update session in db", slog.Any("error", err))
			return
		}

		if err := c.WriteJSON(session); err != nil {
			g.logger.Error("unable to write json", slog.Any("error", err))
			break
		}
		g.logger.Debug("\t< <session data>")
	}
}
