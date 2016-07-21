package messaging

import (
	"errors"
	"log"

	"github.com/IMQS/messaging/clickatell"
)

// ClickatellSendSMS implements the SendSMS method and converts
// Clickatell specific formats to the generic SMS structures.
func (m message) ClickatellSendSMS() SMSResponse {
	log.Println("Sending message with Clickatell")

	rest := clickatell.Rest(m.Provider.Token, nil)
	cm := clickatell.Message{
		Destination: m.Destination,
		Body:        m.Text,
		ClientMsgId: m.ProviderID,
		From:        m.From,
	}
	resp, err := rest.Send(cm)
	if err != nil {
		err = getError(resp)
	}

	var msRess []SendSMSResponseMessage
	for i := range resp.Data.Message {
		msRes := SendSMSResponseMessage{
			To:        resp.Data.Message[i].To,
			MessageID: resp.Data.Message[i].MessageId,
			ErrorCode: resp.Data.Message[i].Error.Code,
			ErrorDesc: resp.Data.Message[i].Error.Description,
		}
		msRess = append(msRess, msRes)
	}
	smsResponse := SMSResponse{
		Error: err,
		Data:  msRess,
	}

	return smsResponse
}

// ClickatellGetStatus retrieves the delivery status of a mobile number
// using the Clickatell service.
func (m message) ClickatellGetStatus() SMSResponse {
	log.Println("Getting status with ClickatellGetStatus")

	rest := clickatell.Rest(Config.SMSProvider.Token, nil)
	st, err := rest.GetStatus(m.ProviderID)
	if err != nil {
		return SMSResponse{Error: errors.New("ClickatellGetStatus: Could not retrieve status")}
	}

	var msRess []SendSMSResponseMessage
	msRess = append(msRess, SendSMSResponseMessage{
		MessageID: st.Data.APIMessageID,
		ErrorCode: clickatellMapCode(st.Data.StatusCode),
		ErrorDesc: st.Data.Description,
		Quantity:  st.Data.Charge,
	})

	sr := SMSResponse{
		Error: nil,
		Data:  msRess,
	}

	return sr
}

///////////////////////////////////////////////////////////////////////////////

func clickatellMapCode(c string) string {
	switch {
	case c == "004":
		return "delivered"
	case c == "007":
		return "failed"
	}
	return "sent" // no final status available
}

// We're not getting a proper error response from Clickatell.  Attempt to get
// the correct error by looking at the error in the first message.
func getError(r *clickatell.SendResponse) error {
	mErrCode := r.Data.Message[0].Error.Code
	mErrDesc := r.Data.Message[0].Error.Description
	var err error
	if mErrCode != "" {
		err = errors.New(mErrCode + ": " + mErrDesc)
	}
	return err
}
