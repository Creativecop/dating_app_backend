package chat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/response"
	"github.com/neoscoder/aura-backend/internal/restriction"
)

type Authenticator interface {
	Authenticate(ctx context.Context, rawToken string) (*auth.AuthenticatedUser, error)
}

type Handler struct {
	service            *Service
	hub                *Hub
	authenticator      Authenticator
	restrictionChecker RestrictionChecker
}

type RestrictionChecker interface {
	CanPerform(ctx context.Context, userID uint64, action string) error
}

func NewHandler(service *Service, hub *Hub, authenticator Authenticator) *Handler {
	return &Handler{service: service, hub: hub, authenticator: authenticator}
}

func (h *Handler) SetRestrictionChecker(checker RestrictionChecker) {
	h.restrictionChecker = checker
}

func (h *Handler) List(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.List(c.Request.Context(), user.UserID, c.Query("limit"), c.Query("cursor"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Chats fetched successfully", result)
}

func (h *Handler) Messages(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.Messages(c.Request.Context(), user.UserID, c.Param("conversationUuid"), c.Query("limit"), c.Query("cursor"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Messages fetched successfully", result)
}

func (h *Handler) SendMessage(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, created, err := h.service.SendMessage(c.Request.Context(), user.UserID, c.Param("conversationUuid"), req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	response.Success(c, status, "Message sent successfully", result)
}

func (h *Handler) MarkRead(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	var req MarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.MarkRead(c.Request.Context(), user.UserID, c.Param("conversationUuid"), req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Conversation marked as read successfully", result)
}

func (h *Handler) DeleteMessage(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.DeleteMessage(c.Request.Context(), user.UserID, c.Param("messageUuid"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Message deleted successfully", result)
}

func (h *Handler) WebSocket(c *gin.Context) {
	token := websocketToken(c)
	if token == "" {
		response.Unauthorized(c, "Unauthorized.")
		return
	}
	user, err := h.authenticator.Authenticate(c.Request.Context(), token)
	if err != nil {
		response.Error(c, auth.PublicStatusCode(err), auth.PublicErrorMessage(err), auth.PublicErrorCode(err), nil)
		return
	}
	if h.restrictionChecker != nil {
		if err := h.restrictionChecker.CanPerform(c.Request.Context(), user.UserID, restriction.ActionSocketConnect); err != nil {
			response.Error(c, auth.PublicStatusCode(err), auth.PublicErrorMessage(err), auth.PublicErrorCode(err), nil)
			return
		}
	}

	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	client := NewClient(user.UserID, conn)
	h.hub.Register(client)
	defer h.hub.Unregister(client)

	go client.WritePump(context.Background())

	for {
		var incoming IncomingEvent
		if err := wsjson.Read(c.Request.Context(), conn, &incoming); err != nil {
			return
		}
		h.handleSocketEvent(c.Request.Context(), client, user.UserID, incoming)
	}
}

func (h *Handler) handleSocketEvent(ctx context.Context, client *Client, userID uint64, incoming IncomingEvent) {
	switch incoming.Event {
	case ClientEventDeliveredAck:
		var req DeliveredAckRequest
		if err := json.Unmarshal(incoming.Data, &req); err != nil {
			client.send <- errorEvent(CodeValidation, "Invalid delivered acknowledgement payload")
			return
		}
		if err := h.service.MarkDelivered(ctx, userID, req); err != nil {
			client.send <- errorEventFor(err)
		}
	case ClientEventTypingStart, ClientEventTypingStop:
		var req TypingRequest
		if err := json.Unmarshal(incoming.Data, &req); err != nil {
			client.send <- errorEvent(CodeValidation, "Invalid typing payload")
			return
		}
		if err := h.service.Typing(ctx, userID, req.ConversationUUID, incoming.Event == ClientEventTypingStart); err != nil {
			client.send <- errorEventFor(err)
		}
	case ClientEventMarkRead:
		var req struct {
			ConversationUUID string `json:"conversationUuid"`
			MarkReadRequest
		}
		if err := json.Unmarshal(incoming.Data, &req); err != nil {
			client.send <- errorEvent(CodeValidation, "Invalid mark read payload")
			return
		}
		if _, err := h.service.MarkRead(ctx, userID, req.ConversationUUID, req.MarkReadRequest); err != nil {
			client.send <- errorEventFor(err)
		}
	case ClientEventSendMessage:
		client.send <- errorEvent(CodeUnsupportedEvent, "Send messages through the REST API in Phase 9")
	default:
		client.send <- errorEvent(CodeValidation, "Unsupported event")
	}
}

func (h *Handler) writeError(c *gin.Context, err error) {
	if serviceErr, ok := AsServiceError(err); ok {
		response.Error(c, serviceErr.Status, serviceErr.Message, serviceErr.Code, serviceErr.Details)
		return
	}
	response.Internal(c)
}

func currentUser(c *gin.Context) (auth.AuthenticatedUser, bool) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized.")
		return auth.AuthenticatedUser{}, false
	}
	return user, true
}

func websocketToken(c *gin.Context) string {
	raw := c.GetHeader("Authorization")
	if strings.HasPrefix(raw, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	}
	return strings.TrimSpace(c.Query("token"))
}

func errorEvent(code string, message string) Event {
	return Event{Event: EventError, Data: ErrorEventData{Code: code, Message: message}}
}

func errorEventFor(err error) Event {
	if serviceErr, ok := AsServiceError(err); ok {
		return errorEvent(serviceErr.Code, serviceErr.Message)
	}
	if errors.Is(err, io.EOF) {
		return errorEvent(CodeValidation, "Invalid event payload")
	}
	return errorEvent("INTERNAL_SERVER_ERROR", "Internal server error")
}
