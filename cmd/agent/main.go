package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/url"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nfnt/resize"
	"github.com/pion/webrtc/v4"

	"screen-hub/internal/capture"
	"screen-hub/internal/input"
	"screen-hub/internal/proto"
)

var (
	inputHandler input.Handler
	agentID      string
	displayIdx   int
)

func main() {
	serverAddr := flag.String("server", "ws://localhost:8080", "server WebSocket address")
	name := flag.String("name", "", "display name (default: hostname)")
	display := flag.Int("display", 0, "display index to capture")
	flag.Parse()

	displayIdx = *display

	if *name == "" {
		h, _ := os.Hostname()
		*name = h
	}

	inputHandler = input.New()

	for {
		if err := run(*serverAddr, *name); err != nil {
			log.Printf("connection error: %v, reconnecting in 3s...", err)
			time.Sleep(3 * time.Second)
		}
	}
}

func run(serverAddr, name string) error {
	u, err := url.Parse(serverAddr)
	if err != nil {
		return err
	}
	u.Path = "/ws/agent"

	log.Printf("connecting to %s", u.String())
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	bounds := capture.DisplayBounds(displayIdx)

	// Register
	conn.WriteJSON(proto.WSMessage{
		Type:   "register",
		Name:   name,
		OS:     runtime.GOOS,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	})

	// Wait for registration confirmation
	var regMsg proto.WSMessage
	if err := conn.ReadJSON(&regMsg); err != nil {
		return fmt.Errorf("register: %w", err)
	}
	agentID = regMsg.AgentID
	log.Printf("registered as %s (id: %s)", name, agentID)

	// Start thumbnail sender
	stopThumb := make(chan struct{})
	go sendThumbnails(conn, stopThumb)

	// Active WebRTC sessions
	var sessionsMu sync.Mutex
	sessions := make(map[string]*webrtc.PeerConnection)

	defer func() {
		close(stopThumb)
		sessionsMu.Lock()
		for _, pc := range sessions {
			pc.Close()
		}
		sessionsMu.Unlock()
	}()

	for {
		var msg proto.WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return fmt.Errorf("read: %w", err)
		}

		switch msg.Type {
		case "offer":
			go func(browserID, sdp string) {
				pc, err := handleOffer(conn, browserID, sdp)
				if err != nil {
					log.Printf("webrtc offer error: %v", err)
					return
				}
				sessionsMu.Lock()
				sessions[browserID] = pc
				sessionsMu.Unlock()

				pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
					if state == webrtc.PeerConnectionStateClosed ||
						state == webrtc.PeerConnectionStateFailed ||
						state == webrtc.PeerConnectionStateDisconnected {
						sessionsMu.Lock()
						delete(sessions, browserID)
						sessionsMu.Unlock()
						log.Printf("webrtc session ended: %s", browserID)
					}
				})
			}(msg.BrowserID, msg.SDP)

		case "candidate":
			sessionsMu.Lock()
			pc, ok := sessions[msg.BrowserID]
			sessionsMu.Unlock()
			if ok && msg.Candidate != nil {
				var candidate webrtc.ICECandidateInit
				if err := json.Unmarshal(msg.Candidate, &candidate); err == nil {
					pc.AddICECandidate(candidate)
				}
			}
		}
	}
}

func handleOffer(wsConn *websocket.Conn, browserID, sdp string) (*webrtc.PeerConnection, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	// Send ICE candidates to browser via signaling server
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		candidateJSON, _ := json.Marshal(c.ToJSON())
		wsConn.WriteJSON(proto.WSMessage{
			Type:      "candidate",
			BrowserID: browserID,
			Candidate: candidateJSON,
		})
	})

	// Handle data channels created by browser
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		switch dc.Label() {
		case "video":
			dc.OnOpen(func() {
				log.Printf("video channel open for %s", browserID)
				go streamScreen(dc)
			})
		case "control":
			dc.OnOpen(func() {
				log.Printf("control channel open for %s", browserID)
				// Send screen info
				bounds := capture.DisplayBounds(displayIdx)
				info, _ := json.Marshal(proto.ControlMessage{
					Type:   "screen_info",
					Width:  bounds.Dx(),
					Height: bounds.Dy(),
				})
				dc.Send(info)
			})
			dc.OnMessage(func(msg webrtc.DataChannelMessage) {
				handleControlMessage(msg.Data)
			})
		}
	})

	// Set remote offer
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	if err := pc.SetRemoteDescription(offer); err != nil {
		pc.Close()
		return nil, err
	}

	// Create answer
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		pc.Close()
		return nil, err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		pc.Close()
		return nil, err
	}

	// Send answer back
	wsConn.WriteJSON(proto.WSMessage{
		Type:      "answer",
		BrowserID: browserID,
		SDP:       answer.SDP,
	})

	return pc, nil
}

func streamScreen(dc *webrtc.DataChannel) {
	ticker := time.NewTicker(time.Second / 15) // 15 FPS
	defer ticker.Stop()

	for range ticker.C {
		if dc.ReadyState() != webrtc.DataChannelStateOpen {
			return
		}

		img, err := capture.CaptureScreen(displayIdx)
		if err != nil {
			continue
		}

		buf := &bytes.Buffer{}
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 50}); err != nil {
			continue
		}

		if err := dc.Send(buf.Bytes()); err != nil {
			return
		}
	}
}

func handleControlMessage(data []byte) {
	var msg proto.ControlMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	bounds := capture.DisplayBounds(displayIdx)
	// Convert normalized coordinates (0-1) to absolute screen coordinates
	absX := int(msg.X * float64(bounds.Dx()))
	absY := int(msg.Y * float64(bounds.Dy()))

	switch msg.Type {
	case "mousemove":
		inputHandler.MoveMouse(absX, absY)
	case "mousedown":
		inputHandler.MouseDown(absX, absY, msg.Button)
	case "mouseup":
		inputHandler.MouseUp(absX, absY, msg.Button)
	case "scroll":
		inputHandler.Scroll(absX, absY, int(msg.DeltaX), int(msg.DeltaY))
	case "keydown":
		inputHandler.KeyDown(msg.Code)
	case "keyup":
		inputHandler.KeyUp(msg.Code)
	}
}

func sendThumbnails(conn *websocket.Conn, stop chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			img, err := capture.CaptureScreen(displayIdx)
			if err != nil {
				continue
			}

			// Resize to thumbnail
			thumb := resize.Resize(320, 0, img, resize.Lanczos3)

			buf := &bytes.Buffer{}
			jpeg.Encode(buf, thumb, &jpeg.Options{Quality: 40})

			b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
			conn.WriteJSON(proto.WSMessage{
				Type: "thumbnail",
				Data: b64,
			})
		}
	}
}

// Ensure image.RGBA implements image.Image (used by resize).
var _ image.Image = (*image.RGBA)(nil)
