package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type apiClient struct {
	token string
	http  *http.Client
}

type apiChat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
}

type apiChatMember struct {
	Status            string `json:"status"`
	CanDeleteMessages bool   `json:"can_delete_messages,omitempty"`
}

func NewAPIClient(token string) *apiClient {
	return &apiClient{token: token, http: &http.Client{Timeout: 10 * time.Second}}
}

func (a *apiClient) call(ctx context.Context, method string, values url.Values, out any) error {
	endpoint := "https://api.telegram.org/bot" + a.token + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var envelope struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Description string          `json:"description,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}
	if !envelope.OK {
		return fmt.Errorf("telegram %s: %s", method, envelope.Description)
	}
	if out == nil || len(envelope.Result) == 0 {
		return nil
	}
	return json.Unmarshal(envelope.Result, out)
}

func (a *apiClient) GetChat(ctx context.Context, username string) (apiChat, error) {
	v := url.Values{}
	v.Set("chat_id", username)
	var result apiChat
	return result, a.call(ctx, "getChat", v, &result)
}

func (a *apiClient) IsAdmin(ctx context.Context, chatID, userID int64) (bool, error) {
	v := url.Values{}
	v.Set("chat_id", strconv.FormatInt(chatID, 10))
	v.Set("user_id", strconv.FormatInt(userID, 10))
	var member apiChatMember
	if err := a.call(ctx, "getChatMember", v, &member); err != nil {
		return false, err
	}
	return member.Status == "administrator" || member.Status == "creator", nil
}
