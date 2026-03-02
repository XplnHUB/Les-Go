package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
	WSConn     *websocket.Conn
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

type authResponse struct {
	Token string `json:"token"`
}

type keyResponse struct {
	PublicKey string `json:"public_key"`
}

func (c *APIClient) Login(username, pass string) error {
	payload := map[string]string{"username": username, "password": pass}
	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", c.BaseURL+"/api/login", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s", string(body))
	}

	var res authResponse
	json.NewDecoder(resp.Body).Decode(&res)
	c.Token = res.Token

	return nil
}

func (c *APIClient) Register(username, pass, pubKey string) error {
	payload := map[string]string{
		"username":   username,
		"password":   pass,
		"public_key": pubKey,
	}
	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", c.BaseURL+"/api/register", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed: %s", string(body))
	}

	return nil
}

func (c *APIClient) GetPublicKey(username string) (string, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+"/api/users/"+username+"/key", nil)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get public key, status: %d", resp.StatusCode)
	}

	var res keyResponse
	json.NewDecoder(resp.Body).Decode(&res)
	return res.PublicKey, nil
}

func (c *APIClient) ConnectWebSocket() error {
	if c.Token == "" {
		return fmt.Errorf("not authenticated")
	}

	wsURL := strings.Replace(c.BaseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1) + "/ws?token=" + c.Token

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}
	c.WSConn = conn
	return nil
}

func (c *APIClient) SendMessage(to, encryptedPayload string) error {
	payload := map[string]string{
		"to":             to,
		"encrypted_data": encryptedPayload,
	}
	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", c.BaseURL+"/api/messages/send", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send message: %s", string(body))
	}
	return nil
}

func (c *APIClient) FetchUnreadMessages() ([]map[string]interface{}, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+"/api/messages/unread", nil)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch unread")
	}

	var res []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	return res, nil
}
