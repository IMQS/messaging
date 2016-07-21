package messaging

import (
	"errors"
	"log"
)

type message struct {
	ID          string      // The internal ID from the database
	ProviderID  string      // The ID assigned by the SMS provider in the send response
	Destination []string    // List of string mobile numbers to send to
	Text        string      // The text message to send
	From        string      // Optional, provide a description for the sender
	Provider    smsProvider // SMS Provider to send the messages through
}

// SMSResponse struct represents the response of a "send"
// API call.
type SMSResponse struct {
	Error error
	Data  []SendSMSResponseMessage
}

// SendSMSResponseMessage struct represents the response of a message contained
// within a "send" API call.
type SendSMSResponseMessage struct {
	To        string
	MessageID string
	ErrorCode string
	ErrorDesc string
	Quantity  int
}

// SendSMS implements REST APIs for SMS providers, as configured in the config.
// It also stores all messages in a DB for later reference
func SendSMS(msg, eml string, ns []string) (string, error) {
	log.Printf("User %v sending message '%v' to %v recipients.", eml, msg, len(ns))

	if !Config.SMSProvider.Enabled {
		return "", errors.New("SendSMS disabled in config, not sending")
	}

	sendID, err := splitBatchAndSend(msg, eml, ns) // Split message into batches if required by provider

	return sendID, err
}

// GetNumberStatus retrieves the delivery status of the last-sent message to a specific MSISDN
func GetNumberStatus(n string) (string, error) {
	mID, sendLogID, st, err := DB.getLastSMSID(n)
	if err != nil {
		return "", err
	}

	if st != "0" && st != "sent" {
		return st, nil
	}

	stDesc, err := getStatus(mID, sendLogID)
	return stDesc, err
}

func getStatus(apiID, sendLogID string) (string, error) {
	resp := callMethod(message{ProviderID: apiID}, Config.SMSProvider.Name+"GetStatus").(SMSResponse)
	if resp.Error != nil {
		return "", resp.Error
	}
	stCode := resp.Data[0].ErrorCode
	stDesc := resp.Data[0].ErrorDesc

	if err := DB.updateSMSData(apiID, stCode, stDesc, sendLogID); err != nil {
		return "", errors.New("SendSMS DB error")
	}

	return stCode, nil
}

// UpdateStatus is executed on an interval, finding all unresolved delivery
// statusses from the last interval and retrieving it from the service provider
func UpdateStatus(i int) {
	log.Printf("UpdateStatus running on interval: %v", i)
	aIDs, err := DB.getUnresolvedIDs(i)

	if err != nil {
		log.Printf("UpdateStatus failed: %v", err)
	}
	for x := 0; x < len(aIDs); x++ {
		getStatus(aIDs[x][0], aIDs[x][1])
	}

}

func splitBatchAndSend(msg, eml string, ns []string) (string, error) {
	var err error
	var sendID string
	bs := Config.SMSProvider.MaxBatchSize
	if bs > 0 && len(ns) > bs {
		sendID, err = sendSMSBatch(msg, eml, ns[:bs])
		splitBatchAndSend(msg, eml, ns[bs:])
	} else {
		sendID, err = sendSMSBatch(msg, eml, ns)
	}
	return sendID, err
}

func sendSMSBatch(msg, eml string, ns []string) (string, error) {

	m := message{
		Destination: ns,
		Text:        msg,
		//ClientMsgId: "0",
		From:     "IMQS",
		Provider: Config.SMSProvider,
	}
	resp := callMethod(m, Config.SMSProvider.Name+"SendSMS").(SMSResponse)

	sendID, err := DB.createSMSData(msg, eml, &resp)
	if err != nil {
		return "", errors.New("SendSMS DB error")
	}

	return sendID, resp.Error
}
