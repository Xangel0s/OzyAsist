package google

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient}
}

type DraftRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type DraftResponse struct {
	ID      string `json:"id"`
	Message struct {
		ID       string `json:"id"`
		ThreadID string `json:"threadId"`
	} `json:"message"`
}

type SearchMessagesResponse struct {
	Messages []struct {
		ID       string `json:"id"`
		ThreadID string `json:"threadId"`
	} `json:"messages"`
	ResultSizeEstimate int `json:"resultSizeEstimate"`
}

// SearchGmail busca mensajes en la bandeja de entrada según una consulta
func (c *Client) SearchGmail(ctx context.Context, accessToken, query string) (*SearchMessagesResponse, error) {
	endpoint := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages?q=%s&maxResults=10", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error buscando en gmail (status %d): %s", resp.StatusCode, string(body))
	}

	var searchResp SearchMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}
	return &searchResp, nil
}

// CreateGmailDraft crea un borrador seguro sin enviar el correo
func (c *Client) CreateGmailDraft(ctx context.Context, accessToken string, draft DraftRequest) (*DraftResponse, error) {
	rawMessage := fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		draft.To, draft.Subject, draft.Body)
	encodedRaw := base64.RawURLEncoding.EncodeToString([]byte(rawMessage))

	payload := map[string]interface{}{
		"message": map[string]string{
			"raw": encodedRaw,
		},
	}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://gmail.googleapis.com/gmail/v1/users/me/drafts", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creando borrador en gmail (status %d): %s", resp.StatusCode, string(respBody))
	}

	var draftResp DraftResponse
	if err := json.NewDecoder(resp.Body).Decode(&draftResp); err != nil {
		return nil, err
	}
	return &draftResp, nil
}

// ListCalendarEvents consulta eventos de la agenda principal
func (c *Client) ListCalendarEvents(ctx context.Context, accessToken string, maxResults int) (string, error) {
	if maxResults <= 0 {
		maxResults = 5
	}
	endpoint := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/primary/events?maxResults=%d&orderBy=startTime&singleEvents=true", maxResults)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
