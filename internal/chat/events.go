package chat

import "encoding/json"

type Event struct {
	Event string `json:"event"`
	Data  any    `json:"data,omitempty"`
}

type IncomingEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data,omitempty"`
}

type ErrorEventData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
