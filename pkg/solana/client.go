package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"pyrolytics/config"
	"github.com/gorilla/websocket"
)

type Client struct {
	cfg   *config.SolanaConfig
	conn  *websocket.Conn
	mu    sync.Mutex
	done  chan struct{}
	subs  map[int]chan json.RawMessage
	subID int
}

type WSRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type WSResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *WSError        `json:"error,omitempty"`
}

type WSError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewClient(cfg *config.SolanaConfig) (*Client, error) {
	return &Client{
		cfg:   cfg,
		done:  make(chan struct{}),
		subs:  make(map[int]chan json.RawMessage),
		subID: 1,
	}, nil
}

func (c *Client) Connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: c.cfg.Timeout,
	}

	conn, _, err := dialer.Dial(c.cfg.WSURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Solana WebSocket: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// Start message handler
	go c.handleMessages()

	return nil
}

func (c *Client) handleMessages() {
	for {
		select {
		case <-c.done:
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			var response WSResponse
			if err := json.Unmarshal(message, &response); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			if response.Error != nil {
				log.Printf("WebSocket error: %v", response.Error)
				continue
			}

			c.mu.Lock()
			if ch, ok := c.subs[response.ID]; ok {
				select {
				case ch <- response.Result:
				default:
					log.Printf("Subscription channel full for ID %d", response.ID)
				}
			}
			c.mu.Unlock()
		}
	}
}

func (c *Client) Subscribe(ctx context.Context, method string, params []interface{}) (chan json.RawMessage, error) {
	c.mu.Lock()
	id := c.subID
	c.subID++
	c.mu.Unlock()

	ch := make(chan json.RawMessage, 100)
	c.mu.Lock()
	c.subs[id] = ch
	c.mu.Unlock()

	req := WSRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.conn.WriteJSON(req); err != nil {
		c.mu.Lock()
		delete(c.subs, id)
		c.mu.Unlock()
		close(ch)
		return nil, fmt.Errorf("failed to send subscription request: %w", err)
	}

	return ch, nil
}

func (c *Client) Close() error {
	close(c.done)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
