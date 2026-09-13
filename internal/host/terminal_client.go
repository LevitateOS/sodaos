package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// Terminal is a private helper stream. Only an authorized attachment sends
// heartbeats; no web owner renews the native shell's lifetime.
type Terminal struct{ conn *websocket.Conn }

func (c *Client) OpenTerminal(ctx context.Context, in TerminalRequest) (*Terminal, error) {
	if !in.valid(time.Now()) {
		return nil, errors.New("invalid terminal request")
	}
	httpClient := *c.HTTP
	httpClient.Timeout = 0 // Stream lifetime is explicit, not the buffered RPC timeout.
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(dialCtx, "ws://soda-host/terminal", &websocket.DialOptions{HTTPClient: &httpClient})
	if err != nil {
		return nil, errors.New("native terminal unavailable")
	}
	conn.SetReadLimit(terminalFrameLimit)
	body, _ := json.Marshal(in)
	if err = conn.Write(dialCtx, websocket.MessageText, body); err != nil {
		conn.CloseNow()
		return nil, errors.New("native terminal unavailable")
	}
	return &Terminal{conn: conn}, nil
}
func (t *Terminal) Send(ctx context.Context, f TerminalFrame) error {
	if !f.inputValid() {
		return errors.New("invalid terminal control")
	}
	if err := writeTerminal(ctx, t.conn, f); err != nil {
		return errors.New("native terminal transport ended")
	}
	return nil
}
func (t *Terminal) Receive(ctx context.Context) (TerminalFrame, error) {
	kind, body, err := t.conn.Read(ctx)
	var f TerminalFrame
	if err != nil || kind != websocket.MessageText || strictjson.Decode(bytes.NewReader(body), &f) != nil || !f.outputValid() {
		return f, errors.New("native terminal transport ended")
	}
	return f, nil
}
func (t *Terminal) Close() { _ = t.conn.CloseNow() }

// TerminalStates performs one bounded operation over the existing private wire.
// A missing/unavailable reply is not absence and never triggers Create or End.
func (c *Client) TerminalStates(ctx context.Context, in TerminalRequest) ([]TerminalState, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	in.Expires = time.Now().Add(30 * time.Second).Unix()
	if in.Action == "attach" {
		return nil, errors.New("not a terminal observation")
	}
	t, err := c.OpenTerminal(ctx, in)
	if err != nil {
		return nil, err
	}
	defer t.Close()
	f, err := t.Receive(ctx)
	if err != nil || f.Type != "metadata" || f.Terminals == nil {
		return nil, errors.New("native terminal outcome unavailable")
	}
	if in.Action != "list" && (len(*f.Terminals) > 1 || len(*f.Terminals) == 1 && (*f.Terminals)[0].ID != in.ID) {
		return nil, errors.New("native terminal target changed")
	}
	return *f.Terminals, nil
}
