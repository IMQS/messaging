package messaging

import (
	"errors"

	"github.com/IMQS/messaging/clickatell"
)

type ClickatellSender struct {
}

// SendSMS implements the SendSMS method and converts
// Clickatell specific formats to the generic SMS structures.
func (c ClickatellSender) SendSMS(s *MessagingServer, m message) ([]SendSMSResponseMessage, error) {
	rest := clickatell.Rest(m.Provider.Token, nil)
	cm := clickatell.Message{
		Destination: m.Destination,
		Body:        m.Text,
		ClientMsgId: m.ProviderID,
		From:        m.From,
	}
	resp, err := rest.Send(cm)
	if err == nil {
		err = getError(resp, err)
	}

	var msRess []SendSMSResponseMessage
	for _, dm := range resp.Data.Message {
		msRes := SendSMSResponseMessage{
			To:        dm.To,
			MessageID: dm.MessageId,
			ErrorCode: dm.Error.Code,
			ErrorDesc: dm.Error.Description,
		}
		msRess = append(msRess, msRes)
	}

	return msRess, err
}

// GetStatus retrieves the delivery status of a mobile number
// using the Clickatell service.
func (c ClickatellSender) GetStatus(s *MessagingServer, m message) ([]SendSMSResponseMessage, error) {
	rest := clickatell.Rest(s.Config.SMSProvider.Token, nil)
	st, err := rest.GetStatus(m.ProviderID)
	var msRess []SendSMSResponseMessage
	if err != nil {
		return msRess, errors.New("ClickatellGetStatus: Could not retrieve status")
	}

	msRess = append(msRess, SendSMSResponseMessage{
		MessageID: st.Data.APIMessageID,
		ErrorCode: clickatellMapCode(st.Data.StatusCode),
		ErrorDesc: st.Data.Description,
		Segments:  st.Data.Charge,
	})

	return msRess, nil
}

///////////////////////////////////////////////////////////////////////////////

func clickatellMapCode(c string) string {
	switch {
	case c == "004":
		return Delivered
	case c == "007":
		return Failed
	}
	return Sent // no final status available
}

// We're not getting a proper error response from Clickatell.  Attempt to get
// the correct error by looking at the error in the first message.
func getError(r *clickatell.SendResponse, e error) error {
	if len(r.Data.Message) <= 0 {
		return e
	}

	mErrCode := r.Data.Message[0].Error.Code
	mErrDesc := r.Data.Message[0].Error.Description
	var err error
	if mErrCode != "" {
		err = errors.New(mErrCode + ": " + mErrDesc)
	}
	return err
}
