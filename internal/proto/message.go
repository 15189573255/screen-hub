package proto

import "encoding/json"

// WSMessage is the unified WebSocket message format between agent, server, and browser.
type WSMessage struct {
	Type      string          `json:"type"`
	Name      string          `json:"name,omitempty"`
	OS        string          `json:"os,omitempty"`
	Width     int             `json:"width,omitempty"`
	Height    int             `json:"height,omitempty"`
	Data      string          `json:"data,omitempty"`
	AgentID   string          `json:"agent_id,omitempty"`
	BrowserID string          `json:"browser_id,omitempty"`
	SDP       string          `json:"sdp,omitempty"`
	Candidate json.RawMessage `json:"candidate,omitempty"`
	Agents    []AgentInfo     `json:"agents,omitempty"`
	Agent     *AgentInfo      `json:"agent,omitempty"`
}

type AgentInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	OS     string `json:"os"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// ControlMessage is sent over WebRTC data channel for input events.
type ControlMessage struct {
	Type   string  `json:"type"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Button int     `json:"button,omitempty"`
	DeltaX float64 `json:"deltaX,omitempty"`
	DeltaY float64 `json:"deltaY,omitempty"`
	Key    string  `json:"key,omitempty"`
	Code   string  `json:"code,omitempty"`
	Width  int     `json:"width,omitempty"`
	Height int     `json:"height,omitempty"`
}
