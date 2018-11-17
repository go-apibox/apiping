package apiping

import (
	"net/http"
	"strings"

	"github.com/go-apibox/api"
	"github.com/gorilla/websocket"
)

type WsHandler func(conn *websocket.Conn) error

type Ping struct {
	app       *api.App
	disabled  bool
	wsEnabled bool
	wsHandler WsHandler
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func NewPing(app *api.App) *Ping {
	cfg := app.Config
	disabled := cfg.GetDefaultBool("apiping.disabled", false)
	wsEnabled := cfg.GetDefaultBool("apiping.websocket_enabled", false)

	ping := new(Ping)
	ping.app = app
	ping.disabled = disabled
	ping.wsEnabled = wsEnabled
	return ping
}

func (p *Ping) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if p.disabled {
		next(w, r)
		return
	}

	c, err := api.NewContext(p.app, w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	action := c.Input.GetAction()
	if action == "Ping" {
		if strings.ToLower(r.Header.Get("Upgrade")) != "websocket" {
			api.WriteResponse(c, nil)
		} else if p.wsEnabled {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			if p.wsHandler != nil {
				if err = p.wsHandler(conn); err != nil {
					return
				}
			} else {
				if err = defaultWsHandler(conn); err != nil {
					return
				}
			}
		}
		return
	}

	next(w, r)
}

func (p *Ping) SetWsHandler(handler WsHandler) {
	p.wsHandler = handler
}

func defaultWsHandler(conn *websocket.Conn) error {
	for {
		messageType, pBytes, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if err = conn.WriteMessage(messageType, pBytes); err != nil {
			return err
		}
	}
	return nil
}
