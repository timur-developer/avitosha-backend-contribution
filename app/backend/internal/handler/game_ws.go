package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/realtime"
)

const (
	webSocketWriteTimeout = 5 * time.Second
	webSocketPongTimeout  = 60 * time.Second
	webSocketPingInterval = 25 * time.Second
)

type GameWebSocketHandler struct {
	logger   *slog.Logger
	hub      realtime.EventSubscriber
	upgrader websocket.Upgrader
}

func NewGameWebSocketHandler(
	logger *slog.Logger,
	hub realtime.EventSubscriber,
	frontendOrigin string,
) GameWebSocketHandler {
	if logger == nil {
		logger = slog.Default()
	}
	frontendOrigin = strings.TrimSpace(frontendOrigin)
	return GameWebSocketHandler{
		logger: logger, hub: hub,
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 5 * time.Second,
			CheckOrigin: func(r *http.Request) bool {
				origin := strings.TrimSpace(r.Header.Get("Origin"))
				return origin == "" || (frontendOrigin != "" && origin == frontendOrigin)
			},
		},
	}
}

func (handler GameWebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := gameUserID(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
		return
	}
	if handler.hub == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "realtime_unavailable", "Realtime updates are unavailable")
		return
	}

	connection, err := handler.upgrader.Upgrade(w, r, nil)
	if err != nil {
		handler.logger.Warn("websocket upgrade failed", "user_id", userID, "error", err.Error())
		return
	}
	defer func() {
		if err := connection.Close(); err != nil {
			handler.logger.Debug("websocket close failed", "user_id", userID, "error", err.Error())
		}
	}()

	subscription := handler.hub.Subscribe(userID)
	defer subscription.Close()
	disconnected := make(chan struct{})
	go readWebSocket(connection, disconnected)

	pingTicker := time.NewTicker(webSocketPingInterval)
	defer pingTicker.Stop()
	for {
		select {
		case events, open := <-subscription.Messages():
			if !open {
				_ = connection.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "client is too slow"),
					time.Now().Add(webSocketWriteTimeout),
				)
				return
			}
			if err := writeWebSocketEvents(connection, events); err != nil {
				return
			}
		case <-pingTicker.C:
			if err := connection.WriteControl(
				websocket.PingMessage, nil, time.Now().Add(webSocketWriteTimeout),
			); err != nil {
				return
			}
		case <-disconnected:
			return
		case <-r.Context().Done():
			return
		}
	}
}

func readWebSocket(connection *websocket.Conn, disconnected chan<- struct{}) {
	defer close(disconnected)
	_ = connection.SetReadDeadline(time.Now().Add(webSocketPongTimeout))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(webSocketPongTimeout))
	})
	for {
		if _, _, err := connection.ReadMessage(); err != nil {
			return
		}
	}
}

func writeWebSocketEvents(connection *websocket.Conn, events []model.DomainEvent) error {
	payload := struct {
		Events []domainEventDTO `json:"events"`
	}{Events: make([]domainEventDTO, len(events))}
	for index, event := range events {
		payload.Events[index] = newDomainEventDTO(event)
	}
	if err := connection.SetWriteDeadline(time.Now().Add(webSocketWriteTimeout)); err != nil {
		return err
	}
	return connection.WriteJSON(payload)
}
