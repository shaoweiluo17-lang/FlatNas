package handlers

import (
	"strings"

	"flatnasgo-backend/config"

	socketio "github.com/googollee/go-socket.io"
	"github.com/golang-jwt/jwt/v5"
)

type MemoUpdatePayload struct {
	Token    string      `json:"token"`
	WidgetId string      `json:"widgetId"`
	Content  interface{} `json:"content"`
}

type TodoUpdatePayload struct {
	Token    string      `json:"token"`
	WidgetId string      `json:"widgetId"`
	Content  interface{} `json:"content"`
}

func BindMemoHandlers(server *socketio.Server) {
	server.OnEvent("/", "memo:update", func(s socketio.Conn, msg interface{}) {
		token, widgetId, content, ok := parseMemoPayload(msg)
		if !ok {
			return
		}
		if _, ok := validateSocketToken(token); !ok {
			return
		}
		server.BroadcastToNamespace("/", "memo:updated", map[string]interface{}{
			"widgetId": widgetId,
			"content":  content,
		})
	})
}

func BindTodoHandlers(server *socketio.Server) {
	server.OnEvent("/", "todo:update", func(s socketio.Conn, msg interface{}) {
		token, widgetId, content, ok := parseTodoPayload(msg)
		if !ok {
			return
		}
		if _, ok := validateSocketToken(token); !ok {
			return
		}
		server.BroadcastToNamespace("/", "todo:updated", map[string]interface{}{
			"widgetId": widgetId,
			"content":  content,
		})
	})
}

func parseMemoPayload(msg interface{}) (string, string, interface{}, bool) {
	switch v := msg.(type) {
	case MemoUpdatePayload:
		if v.WidgetId == "" || v.Content == nil {
			return "", "", nil, false
		}
		return v.Token, v.WidgetId, v.Content, true
	case *MemoUpdatePayload:
		if v == nil || v.WidgetId == "" || v.Content == nil {
			return "", "", nil, false
		}
		return v.Token, v.WidgetId, v.Content, true
	case map[string]interface{}:
		token, _ := v["token"].(string)
		widgetId, _ := v["widgetId"].(string)
		content := v["content"]
		if widgetId == "" || content == nil {
			return "", "", nil, false
		}
		return token, widgetId, content, true
	default:
		return "", "", nil, false
	}
}

func parseTodoPayload(msg interface{}) (string, string, interface{}, bool) {
	switch v := msg.(type) {
	case TodoUpdatePayload:
		if v.WidgetId == "" || v.Content == nil {
			return "", "", nil, false
		}
		return v.Token, v.WidgetId, v.Content, true
	case *TodoUpdatePayload:
		if v == nil || v.WidgetId == "" || v.Content == nil {
			return "", "", nil, false
		}
		return v.Token, v.WidgetId, v.Content, true
	case map[string]interface{}:
		token, _ := v["token"].(string)
		widgetId, _ := v["widgetId"].(string)
		content := v["content"]
		if widgetId == "" || content == nil {
			return "", "", nil, false
		}
		return token, widgetId, content, true
	default:
		return "", "", nil, false
	}
}

func validateSocketToken(tokenStr string) (string, bool) {
	if tokenStr == "" {
		return "", false
	}
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	tok, err := jwt.Parse(
		tokenStr,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(config.GetSecretKeyString()), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || tok == nil || !tok.Valid {
		return "", false
	}
	if claims, ok := tok.Claims.(jwt.MapClaims); ok {
		if username, ok := claims["username"].(string); ok && username != "" {
			return username, true
		}
	}
	return "", false
}
