package main

import (
	"context"
	"fmt"
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

type wsCommand string

const (
	wsNoop    wsCommand = "g"
	wsOpen    wsCommand = "o"
	wsFlag    wsCommand = "f"
	wsChord   wsCommand = "c"
	wsForfeit wsCommand = "r" // =)
)

type gameExecutor struct {
	*application
	*mines.GameState
}

func newGameExecutor(app *application, state *mines.GameState) *gameExecutor {
	return &gameExecutor{app, state}
}

func (game gameExecutor) openCell(args []string) error {
	x, y, err := parseXY(args)
	if err != nil {
		return err
	}
	if !game.PointInBounds(x, y) {
		return fmt.Errorf("invalid square coordinates")
	}
	game.OpenCell(x, y)
	return nil
}

func (game gameExecutor) flagCell(args []string) error {
	x, y, err := parseXY(args)
	if err != nil {
		return err
	}
	if !game.PointInBounds(x, y) {
		return fmt.Errorf("invalid square coordinates")
	}
	game.FlagCell(x, y)
	return nil
}

func (game gameExecutor) chordCell(args []string) error {
	x, y, err := parseXY(args)
	if err != nil {
		return err
	}
	if !game.PointInBounds(x, y) {
		return fmt.Errorf("invalid square coordinates")
	}
	game.ChordCell(x, y)
	return nil
}

func (game gameExecutor) forfeit() error {
	game.Forfeit()
	return nil
}

func (game gameExecutor) execute(query string) error {
	tokens := strings.Split(" ", query)
	cmd, args := wsCommand(tokens[0]), tokens[1:]
	switch cmd {
	case wsNoop:
		return nil
	case wsOpen:
		return game.openCell(args)
	case "f":
		return game.flagCell(args)
	case "c":
		return game.chordCell(args)
	case "r":
		return game.forfeit()
	default:
		return fmt.Errorf("unknown command")
	}
}

func (game gameExecutor) wsRunGameLoop(
	ctx context.Context, conn *websocket.Conn, session *repository.GameSession,
) error {
	for {
		mt, buf, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if mt != websocket.TextMessage {
			return nil
		}

		message := strings.TrimSpace(string(buf))
		lines := strings.Split(message, "\n")
	LINES:
		for _, line := range lines {
			err := game.execute(strings.TrimSpace(line))
			if err != nil {
				return err
			}
			if game.Won || game.Dead {
				session.EndedAt.Time = time.Now().UTC()
				game.RevealPlayerGrid()
				break LINES
			}
		}

		stateBuf, err := game.Bytes()
		if err != nil {
			return fmt.Errorf("unable to serialize game state: %w", err)
		}

		err = game.repo.UpdateGameSession(
			ctx,
			session.GameSessionId,
			repository.UpdateGameSessionParams{
				State:   &stateBuf,
				Dead:    &game.Dead,
				Won:     &game.Won,
				EndedAt: &session.EndedAt.Time,
			})
		if err != nil {
			return fmt.Errorf("unable to update session in db: %w", err)
		}

		if err := conn.WriteJSON(session); err != nil {
			return fmt.Errorf("unable to write json: %w", err)
		}
	}
}

func (app application) wsConnect(w http.ResponseWriter, r *http.Request) {
	sessionId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		app.badRequest(w)
		app.logger.Debug("invalid session id")
		return
	}

	session, err := app.repo.FetchGameSession(r.Context(), sessionId)
	if err != nil {
		if err == pgx.ErrNoRows {
			app.notFound(w)
			app.logger.Debug("session id not found")
		} else {
			app.internalError(w, "could not fetch session from db", slog.Any("error", err))
		}
		return
	}

	state, err := mines.DecodeGameState(session.State)
	if err != nil {
		app.internalError(w, "game state invalid", slog.Any("error", err))
		return
	}

	conn, err := app.ws.Upgrader.Upgrade(w, r, nil) // headers sent here
	if err != nil {
		app.logger.Error("unable to upgrade", slog.Any("error", err))
		return
	}
	defer conn.Close()

	app.logger.Debug("established WS connection")

	game := newGameExecutor(&app, state)
	if err := game.wsRunGameLoop(r.Context(), conn, session); err != nil {
		if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
			return
		}
		app.logger.Warn("error in ws loop", slog.Any("err", err))
		return
	}
}

func parseXY(args []string) (x int, y int, err error) {
	if len(args) != 2 {
		err = fmt.Errorf("invalid args")
		return
	}
	if x, err = strconv.Atoi(args[0]); err != nil {
		err = fmt.Errorf("first argument must be an int")
		return
	}
	if y, err = strconv.Atoi(args[1]); err != nil {
		err = fmt.Errorf("second argument must be an int")
		return
	}
	return
}
