package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"nhooyr.io/websocket"

	"rustymanager/internal/db"
)

// PushSender sends a push notification to all subscribed browsers,
// excluding the user identified by excludeUserID.
type PushSender interface {
	Send(ctx context.Context, title, body, url string, excludeUserID int64)
}

// ChatChannel manages WebSocket clients grouped by project.
type ChatChannel struct {
	mu          sync.RWMutex
	rooms       map[int64]map[*chatClient]bool
	activeCalls map[int64]string // projectID → room name, "" if none
	queries     db.Querier
	push        PushSender
}

func NewChatChannel(q db.Querier, ps PushSender) *ChatChannel {
	return &ChatChannel{
		rooms:       make(map[int64]map[*chatClient]bool),
		activeCalls: make(map[int64]string),
		queries:     q,
		push:        ps,
	}
}

type chatClient struct {
	ch        *ChatChannel
	conn      *websocket.Conn
	projectID int64
	send      chan []byte
	cancel    context.CancelFunc
}

// incomingMsg is what the browser sends.
type incomingMsg struct {
	Type           string `json:"type"` // "message" | "call_start" | "call_end"
	UserID         int64  `json:"user_id"`
	Content        string `json:"content"`
	RoomName       string `json:"room_name"` // for call events
	AttachmentURL  string `json:"attachment_url"`
	AttachmentType string `json:"attachment_type"`
}

// outgoingMsg is what the server sends for a single message.
type outgoingMsg struct {
	Type           string `json:"type"`
	ID             int64  `json:"id"`
	UserID         int64  `json:"user_id"`
	UserName       string `json:"user_name"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at"`
	RoomName       string `json:"room_name,omitempty"`
	AttachmentURL  string `json:"attachment_url,omitempty"`
	AttachmentType string `json:"attachment_type,omitempty"`
}

// historyMsg is the initial payload sent on connect.
type historyMsg struct {
	Type     string        `json:"type"`
	Messages []outgoingMsg `json:"messages"`
}

func (ch *ChatChannel) register(c *chatClient) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if ch.rooms[c.projectID] == nil {
		ch.rooms[c.projectID] = make(map[*chatClient]bool)
	}
	ch.rooms[c.projectID][c] = true
}

func (ch *ChatChannel) unregister(c *chatClient) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if room, ok := ch.rooms[c.projectID]; ok {
		delete(room, c)
		if len(room) == 0 {
			delete(ch.rooms, c.projectID)
		}
	}
}

func (ch *ChatChannel) broadcast(projectID int64, msg []byte) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	for c := range ch.rooms[projectID] {
		select {
		case c.send <- msg:
		default:
			// slow client — drop message
		}
	}
}

// HandleWS upgrades the connection and starts the read/write pumps.
func (ch *ChatChannel) HandleWS(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}

	conn, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(c.Request().Context())

	client := &chatClient{
		ch:        ch,
		conn:      conn,
		projectID: projectID,
		send:      make(chan []byte, 64),
		cancel:    cancel,
	}
	ch.register(client)

	// Send message history (newest 100, reversed to chronological order).
	rows, err := ch.queries.ListChatMessagesByProject(ctx, projectID)
	if err != nil {
		log.Printf("chat: list history: %v", err)
	} else {
		msgs := make([]outgoingMsg, len(rows))
		for i, r := range rows {
			msgs[len(rows)-1-i] = outgoingMsg{
				Type:           r.MessageType,
				ID:             r.ID,
				UserID:         r.UserID.Int64,
				UserName:       r.UserName,
				Content:        r.Content,
				CreatedAt:      r.CreatedAt.Format(time.RFC3339),
				RoomName:       r.RoomName,
				AttachmentURL:  r.AttachmentUrl,
				AttachmentType: r.AttachmentType,
			}
		}
		if b, err := json.Marshal(historyMsg{Type: "history", Messages: msgs}); err == nil {
			client.send <- b
		}
	}

	// Notify late joiners if a call is active.
	// First check in-memory map; lazy-load from DB if not set (e.g. after restart).
	ch.mu.RLock()
	activeRoom, known := ch.activeCalls[projectID]
	ch.mu.RUnlock()
	if !known {
		// Not in map yet — query DB to find out.
		if row, err := ch.queries.GetActiveCallForProject(ctx, projectID); err == nil && row.MessageType == "call_start" {
			activeRoom = row.RoomName
			ch.mu.Lock()
			ch.activeCalls[projectID] = activeRoom
			ch.mu.Unlock()
		} else {
			// Mark as known-empty so we don't re-query every connect.
			ch.mu.Lock()
			ch.activeCalls[projectID] = ""
			ch.mu.Unlock()
		}
	}
	if activeRoom != "" {
		synth := outgoingMsg{
			Type:     "call_start",
			RoomName: activeRoom,
			Content:  "A call is in progress",
		}
		if b, err := json.Marshal(synth); err == nil {
			client.send <- b
		}
	}

	go client.writePump(ctx)
	client.readPump(ctx)
	return nil
}

const pingPeriod = 30 * time.Second

func (c *chatClient) readPump(ctx context.Context) {
	defer func() {
		c.cancel()
		c.ch.unregister(c)
		close(c.send)
	}()

	for {
		_, raw, err := c.conn.Read(ctx)
		if err != nil {
			break
		}

		var in incomingMsg
		if err := json.Unmarshal(raw, &in); err != nil {
			continue
		}

		switch in.Type {
		case "call_start", "call_end":
			c.ch.handleCallEvent(ctx, c, in)
		default:
			// "message" or legacy (no type field)
			if in.Content == "" && in.AttachmentURL == "" {
				continue
			}
			c.ch.handleChatMessage(ctx, c, in)
		}
	}
}

func (ch *ChatChannel) handleChatMessage(ctx context.Context, c *chatClient, in incomingMsg) {
	var userID sql.NullInt64
	if in.UserID > 0 {
		userID = sql.NullInt64{Int64: in.UserID, Valid: true}
	}

	saved, err := ch.queries.CreateChatMessage(ctx, db.CreateChatMessageParams{
		ProjectID:      c.projectID,
		UserID:         userID,
		Content:        in.Content,
		MessageType:    "message",
		RoomName:       "",
		AttachmentUrl:  in.AttachmentURL,
		AttachmentType: in.AttachmentType,
	})
	if err != nil {
		log.Printf("chat: save message: %v", err)
		return
	}

	userName := "Anonymous"
	if userID.Valid {
		if u, err := ch.queries.GetUser(ctx, userID.Int64); err == nil {
			userName = u.Name
		}
	}

	out := outgoingMsg{
		Type:           "message",
		ID:             saved.ID,
		UserID:         userID.Int64,
		UserName:       userName,
		Content:        saved.Content,
		CreatedAt:      saved.CreatedAt.Format(time.RFC3339),
		AttachmentURL:  saved.AttachmentUrl,
		AttachmentType: saved.AttachmentType,
	}
	b, err := json.Marshal(out)
	if err != nil {
		return
	}
	ch.broadcast(c.projectID, b)

	if ch.push != nil {
		project, err := ch.queries.GetProject(ctx, c.projectID)
		projectName := fmt.Sprintf("project %d", c.projectID)
		if err == nil {
			projectName = project.Name
		}
		title := "[rustymanager] " + projectName
		body := userName + ": " + out.Content
		url := fmt.Sprintf("/projects/%d", c.projectID)
		go ch.push.Send(context.Background(), title, body, url, in.UserID)
	}
}

func (ch *ChatChannel) handleCallEvent(ctx context.Context, c *chatClient, in incomingMsg) {
	if in.RoomName == "" {
		return
	}

	var userID sql.NullInt64
	if in.UserID > 0 {
		userID = sql.NullInt64{Int64: in.UserID, Valid: true}
	}

	content := "started a call"
	if in.Type == "call_end" {
		content = "ended a call"
	}

	saved, err := ch.queries.CreateChatMessage(ctx, db.CreateChatMessageParams{
		ProjectID:   c.projectID,
		UserID:      userID,
		Content:     content,
		MessageType: in.Type,
		RoomName:    in.RoomName,
	})
	if err != nil {
		log.Printf("chat: save call event: %v", err)
		return
	}

	userName := "Anonymous"
	if userID.Valid {
		if u, err := ch.queries.GetUser(ctx, userID.Int64); err == nil {
			userName = u.Name
		}
	}

	// Update in-memory active call map.
	ch.mu.Lock()
	if in.Type == "call_start" {
		ch.activeCalls[c.projectID] = in.RoomName
	} else {
		ch.activeCalls[c.projectID] = ""
	}
	ch.mu.Unlock()

	out := outgoingMsg{
		Type:      in.Type,
		ID:        saved.ID,
		UserID:    userID.Int64,
		UserName:  userName,
		Content:   content,
		CreatedAt: saved.CreatedAt.Format(time.RFC3339),
		RoomName:  in.RoomName,
	}
	b, _ := json.Marshal(out)
	ch.broadcast(c.projectID, b)
}

// HandleHistory returns older messages before a given message ID as JSON.
func (ch *ChatChannel) HandleHistory(c echo.Context) error {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	beforeID, err := strconv.ParseInt(c.QueryParam("before"), 10, 64)
	if err != nil || beforeID <= 0 {
		return echo.ErrBadRequest
	}

	rows, err := ch.queries.ListChatMessagesBefore(c.Request().Context(), db.ListChatMessagesBeforeParams{
		ProjectID: projectID,
		ID:        beforeID,
	})
	if err != nil {
		return err
	}

	msgs := make([]outgoingMsg, len(rows))
	for i, r := range rows {
		msgs[len(rows)-1-i] = outgoingMsg{
			Type:           r.MessageType,
			ID:             r.ID,
			UserID:         r.UserID.Int64,
			UserName:       r.UserName,
			Content:        r.Content,
			CreatedAt:      r.CreatedAt.Format(time.RFC3339),
			RoomName:       r.RoomName,
			AttachmentURL:  r.AttachmentUrl,
			AttachmentType: r.AttachmentType,
		}
	}
	return c.JSON(200, msgs)
}

func (c *chatClient) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.Write(ctx, websocket.MessageText, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.Ping(ctx); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
