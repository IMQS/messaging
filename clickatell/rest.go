package clickatell

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type RestClient struct {
	client   *http.Client
	apiToken string
}

func Rest(apiToken string, client *http.Client) *RestClient {
	if client == nil {
		client = http.DefaultClient
	}

	return &RestClient{
		client:   client,
		apiToken: apiToken,
	}
}

func (c *RestClient) applyHeaders(req *http.Request) *http.Request {
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("Authorization", Concat("Bearer", " ", c.apiToken))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Version", "1")
	req.Header.Add("Accept", "application/json")
	return req
}

func (c *RestClient) Send(in Message) (*SendResponse, error) {
	jsonBody, _ := json.Marshal(in)
	req, _ := http.NewRequest("POST", Concat(apiEndpoint, "rest/message"), bytes.NewBuffer(jsonBody))
	resp, err := c.client.Do(c.applyHeaders(req))
	result := &SendResponse{}

	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(result)
		if err == nil {
			err = result.Error.GetError(resp)
		}
	}
	return result, err
}

func (c *RestClient) GetStatus(messageID string) (*GetStatusResponse, error) {
	req, _ := http.NewRequest("GET", Concat(apiEndpoint, "rest/message/", messageID), nil)
	resp, err := c.client.Do(c.applyHeaders(req))
	result := &GetStatusResponse{}
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(result)
	}
	return result, err
}

func (c *RestClient) GetBalance() (*GetBalanceResponse, error) {
	req, _ := http.NewRequest("GET", Concat(apiEndpoint, "rest/account/balance"), nil)
	resp, err := c.client.Do(c.applyHeaders(req))
	result := &GetBalanceResponse{}
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(result)
		if err == nil {
			err = result.Error.GetError(resp)
		}
	}
	return result, err
}
