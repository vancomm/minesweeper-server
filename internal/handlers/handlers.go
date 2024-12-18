package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func SendJSON(w http.ResponseWriter, v any) (int, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}
	w.Header().Add("Content-Type", "application/json")
	return w.Write(payload)
}

func SendJSONOrLog(w http.ResponseWriter,
	logger *slog.Logger,
	v any,
) {
	_, err := SendJSON(w, v)
	if err != nil {
		logger.Error(
			"failed to send data",
			slog.Any("data", v),
			slog.Any("error", err),
		)
	}
}

func SendMessageOrLog(
	w http.ResponseWriter,
	logger *slog.Logger,
	m string,
) {
	_, err := SendJSON(w, map[string]string{
		"message": m,
	})
	if err != nil {
		logger.Error(
			"failed to send message",
			slog.String("message", m),
			slog.Any("error", err),
		)
	}
}

func SendErrorOrLog(
	w http.ResponseWriter,
	logger *slog.Logger,
	e error,
) {
	_, err := SendJSON(w, map[string]string{
		"error": e.Error(),
	})
	if err != nil {
		logger.Error(
			"failed to send error message",
			slog.Any("sent error", e),
			slog.Any("error", err),
		)
	}
}

func sendJSONOrLog(w http.ResponseWriter, logger *slog.Logger, v any) {
	_, err := SendJSON(w, v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"unable to send response",
			slog.Any("response", v),
			slog.Any("error", err),
		)
	}
}

func wrapError(err error) map[string]string {
	return map[string]string{
		"error": err.Error(),
	}
}
