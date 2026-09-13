package mcp

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"sync/atomic"
)

type Client struct {
	Transport Transport
	Name      string

	nextReqID uint64
	pending   map[string]chan *Response
	mu        sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
}

func NewClient(name string, transport Transport) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		Transport: transport,
		Name:      name,
		pending:   make(map[string]chan *Response),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (c *Client) Start(ctx context.Context) error {
	if err := c.Transport.Start(ctx); err != nil {
		return err
	}

	// Start receive loop
	go c.receiveLoop()

	return nil
}

func (c *Client) Stop() error {
	c.cancel()
	return c.Transport.Close()
}

func (c *Client) receiveLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			msg, err := c.Transport.Receive()
			if err != nil {
				// Log or handle error, might mean connection closed
				return
			}

			// Try to parse as Response
			var resp Response
			if err := json.Unmarshal(msg, &resp); err == nil && resp.ID != "" {
				c.mu.Lock()
				ch, ok := c.pending[resp.ID]
				if ok {
					delete(c.pending, resp.ID)
					ch <- &resp
				}
				c.mu.Unlock()
			}
			// Might also be a notification or a request from the server, but for now we only care about responses
		}
	}
}

func (c *Client) sendRequest(ctx context.Context, method string, params interface{}) (*Response, error) {
	id := strconv.FormatUint(atomic.AddUint64(&c.nextReqID, 1), 10)
	
	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	msgBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan *Response, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	if err := c.Transport.Send(msgBytes); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case resp := <-ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp, nil
	}
}

func (c *Client) Initialize(ctx context.Context) (*InitializeResult, error) {
	params := InitializeParams{
		ProtocolVersion: "2024-11-05", // Standard version
		ClientInfo: Implementation{
			Name:    "OzyAssist",
			Version: "1.0.0",
		},
		Capabilities: ClientCapabilities{
			Roots: &struct{ ListChanged bool `json:"listChanged,omitempty"` }{ListChanged: true},
		},
	}

	resp, err := c.sendRequest(ctx, "initialize", params)
	if err != nil {
		return nil, err
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	// Must send initialized notification
	notif := Notification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	notifBytes, _ := json.Marshal(notif)
	c.Transport.Send(notifBytes)

	return &result, nil
}

func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	resp, err := c.sendRequest(ctx, "tools/list", map[string]string{})
	if err != nil {
		return nil, err
	}

	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return result.Tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, arguments interface{}) (*CallToolResult, error) {
	params := CallToolParams{
		Name:      name,
		Arguments: arguments,
	}

	resp, err := c.sendRequest(ctx, "tools/call", params)
	if err != nil {
		return nil, err
	}

	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
