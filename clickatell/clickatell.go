package clickatell

import (
	"errors"
	"fmt"
	"net/http"
)

const (
	apiEndpoint = "https://api.clickatell.com/"
	userAgent   = "GOClickatell"
)

type Message struct {
	ClientMsgId string   `url:"cliMsgId,omitempty" json:"clientMessageID"`
	Destination []string `url:"-" json:"to"`
	Body        string   `url:"text" json:"text"`
	From        string   `url:"from,omitempty" json:"from"`
}

type GetStatusResponse struct {
	Error ErrorResponse `json:"error"`
	Data  struct {
		Charge          int    `json:"charge"`
		StatusCode      string `json:"messageStatus"`
		Description     string `json:"description"`
		APIMessageID    string `json:"apiMessageId"`
		ClientMessageID string `json:"clientMessageID"`
	} `json:"data"`
}

type SendResponse struct {
	Error ErrorResponse `json:"error"`

	Data struct {
		Message []SendResponseMessage `json:"message"`
	} `json:"data"`
}

type SendResponseMessage struct {
	To        string        `json:"to"`
	MessageId string        `json:"apiMessageId"`
	Error     ErrorResponse `json:"error"`
}

type GetBalanceResponse struct {
	Error ErrorResponse `json:"error"`
	Data  struct {
		Balance float64 `json:"balance,string"`
	} `json:"data"`
}

type ErrorResponse struct {
	Description string `json:"description"`
	Code        string `json:"code"`
}

type ClickatellErr struct {
	error
	Code string
}

func (e *ErrorResponse) HasError() bool {
	return e.Description != ""
}

func (e *ErrorResponse) GetError(r *http.Response) error {
	if e.HasError() {
		return fmt.Errorf("clickatell: %v", e.Description)
	}
	return getSuccessStatus(r)
}

func MakeError(err ErrorResponse) *ClickatellErr {
	return &ClickatellErr{errors.New(err.Description), err.Code}
}

func getSuccessStatus(r *http.Response) error {
	switch r.StatusCode {
	case 200:
		return nil
	case 202:
		return nil
	default:
		return fmt.Errorf("clickatell: %v", r.Status)
	}
}
