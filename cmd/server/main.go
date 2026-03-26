package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"screen-hub/internal/proto"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	mu       sync.RWMutex
	agents   map[string]*AgentConn
	browsers map[string]*BrowserConn
}

type AgentConn struct {
	Info proto.AgentInfo
	Conn *websocket.Conn
	mu   sync.Mutex
}

type BrowserConn struct {
	ID   string
	Conn *websocket.Conn
	mu   sync.Mutex
}

func (ac *AgentConn) Send(msg proto.WSMessage) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.Conn.WriteJSON(msg)
}

func (bc *BrowserConn) Send(msg proto.WSMessage) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.Conn.WriteJSON(msg)
}

var hub = &Hub{
	agents:   make(map[string]*AgentConn),
	browsers: make(map[string]*BrowserConn),
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	webDir := flag.String("web", "./web", "web static files directory")
	flag.Parse()

	http.HandleFunc("/ws/agent", handleAgent)
	http.HandleFunc("/ws/browser", handleBrowser)
	http.Handle("/", http.FileServer(http.Dir(*webDir)))

	log.Printf("Server listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func handleAgent(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("agent upgrade error: %v", err)
		return
	}
	defer conn.Close()

	agentID := uuid.New().String()[:8]
	var ac *AgentConn

	for {
		var msg proto.WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		switch msg.Type {
		case "register":
			info := proto.AgentInfo{
				ID:     agentID,
				Name:   msg.Name,
				OS:     msg.OS,
				Width:  msg.Width,
				Height: msg.Height,
			}
			ac = &AgentConn{Info: info, Conn: conn}
			hub.mu.Lock()
			hub.agents[agentID] = ac
			hub.mu.Unlock()

			// Tell agent its assigned ID
			conn.WriteJSON(proto.WSMessage{Type: "registered", AgentID: agentID})

			// Notify all browsers
			hub.mu.RLock()
			for _, bc := range hub.browsers {
				bc.Send(proto.WSMessage{Type: "agent_joined", Agent: &info})
			}
			hub.mu.RUnlock()
			log.Printf("agent registered: %s (%s, %s)", info.Name, agentID, info.OS)

		case "thumbnail":
			hub.mu.RLock()
			for _, bc := range hub.browsers {
				bc.Send(proto.WSMessage{
					Type:    "thumbnail",
					AgentID: agentID,
					Data:    msg.Data,
				})
			}
			hub.mu.RUnlock()

		case "answer":
			hub.mu.RLock()
			bc, ok := hub.browsers[msg.BrowserID]
			hub.mu.RUnlock()
			if ok {
				bc.Send(proto.WSMessage{
					Type:    "answer",
					AgentID: agentID,
					SDP:     msg.SDP,
				})
			}

		case "candidate":
			hub.mu.RLock()
			bc, ok := hub.browsers[msg.BrowserID]
			hub.mu.RUnlock()
			if ok {
				bc.Send(proto.WSMessage{
					Type:      "candidate",
					AgentID:   agentID,
					Candidate: msg.Candidate,
				})
			}
		}
	}

	// Agent disconnected
	if ac != nil {
		hub.mu.Lock()
		delete(hub.agents, agentID)
		hub.mu.Unlock()

		hub.mu.RLock()
		for _, bc := range hub.browsers {
			bc.Send(proto.WSMessage{Type: "agent_left", AgentID: agentID})
		}
		hub.mu.RUnlock()
		log.Printf("agent disconnected: %s", agentID)
	}
}

func handleBrowser(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("browser upgrade error: %v", err)
		return
	}
	defer conn.Close()

	browserID := uuid.New().String()[:8]
	bc := &BrowserConn{ID: browserID, Conn: conn}

	hub.mu.Lock()
	hub.browsers[browserID] = bc
	hub.mu.Unlock()

	// Send current agent list
	hub.mu.RLock()
	agents := make([]proto.AgentInfo, 0, len(hub.agents))
	for _, ac := range hub.agents {
		agents = append(agents, ac.Info)
	}
	hub.mu.RUnlock()

	conn.WriteJSON(proto.WSMessage{
		Type:      "registered",
		BrowserID: browserID,
	})
	conn.WriteJSON(proto.WSMessage{
		Type:   "agents",
		Agents: agents,
	})

	log.Printf("browser connected: %s", browserID)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg proto.WSMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		msg.BrowserID = browserID

		switch msg.Type {
		case "offer":
			hub.mu.RLock()
			ac, ok := hub.agents[msg.AgentID]
			hub.mu.RUnlock()
			if ok {
				ac.Send(proto.WSMessage{
					Type:      "offer",
					BrowserID: browserID,
					SDP:       msg.SDP,
				})
			} else {
				bc.Send(proto.WSMessage{
					Type: "error",
					Data: fmt.Sprintf("agent %s not found", msg.AgentID),
				})
			}

		case "candidate":
			hub.mu.RLock()
			ac, ok := hub.agents[msg.AgentID]
			hub.mu.RUnlock()
			if ok {
				ac.Send(proto.WSMessage{
					Type:      "candidate",
					BrowserID: browserID,
					Candidate: msg.Candidate,
				})
			}
		}
	}

	hub.mu.Lock()
	delete(hub.browsers, browserID)
	hub.mu.Unlock()
	log.Printf("browser disconnected: %s", browserID)
}
