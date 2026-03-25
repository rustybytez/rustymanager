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

// PushSender sends a push notification to all subscribed browsers.
type PushSender interface {
	Send(ctx context.Context, title, body, url string)
}

// ChatChannel manages WebSocket clients grouped by project.
type ChatChannel struct {
	mu      sync.RWMutex
	rooms   map[int64]map[*chatClient]bool
	queries db.Querier
	push    PushSender
}

func NewChatChannel(q db.Querier, ps PushSender) *ChatChannel {
	return &ChatChannel{
		rooms:   make(map[int64]map[*chatClient]bool),
		queries: q,
		push:    ps,
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
	UserID  int64  `json:"user_id"`
	Content string `json:"content"`
}

// outgoingMsg is what the server sends for a single message.
type outgoingMsg struct {
	Type      string `json:"type"`
	ID        int64  `json:"id"`
	UserName  string `json:"user_name"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
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
				Type:      "message",
				ID:        r.ID,
				UserName:  r.UserName,
				Content:   r.Content,
				CreatedAt: r.CreatedAt.Format(time.RFC3339),
			}
		}
		if b, err := json.Marshal(historyMsg{Type: "history", Messages: msgs}); err == nil {
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
		if err := json.Unmarshal(raw, &in); err != nil || in.Content == "" {
			continue
		}

		var userID sql.NullInt64
		if in.UserID > 0 {
			userID = sql.NullInt64{Int64: in.UserID, Valid: true}
		}

		saved, err := c.ch.queries.CreateChatMessage(ctx, db.CreateChatMessageParams{
			ProjectID: c.projectID,
			UserID:    userID,
			Content:   in.Content,
		})
		if err != nil {
			log.Printf("chat: save message: %v", err)
			continue
		}

		userName := "Anonymous"
		if userID.Valid {
			if u, err := c.ch.queries.GetUser(ctx, userID.Int64); err == nil {
				userName = u.Name
			}
		}

		out := outgoingMsg{
			Type:      "message",
			ID:        saved.ID,
			UserName:  userName,
			Content:   saved.Content,
			CreatedAt: saved.CreatedAt.Format(time.RFC3339),
		}
		b, err := json.Marshal(out)
		if err != nil {
			continue
		}
		c.ch.broadcast(c.projectID, b)

		if c.ch.push != nil {
			project, err := c.ch.queries.GetProject(ctx, c.projectID)
			projectName := fmt.Sprintf("project %d", c.projectID)
			if err == nil {
				projectName = project.Name
			}
			title := "New message in " + projectName
			body := userName + ": " + out.Content
			url := fmt.Sprintf("/projects/%d", c.projectID)
			go c.ch.push.Send(context.Background(), title, body, url)
		}
	}
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
			Type:      "message",
			ID:        r.ID,
			UserName:  r.UserName,
			Content:   r.Content,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
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
