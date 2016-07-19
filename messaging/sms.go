package messaging

import (
	"errors"
	"log"
)

type smsProvider struct {
	Name         string
	Token        string
	MaxBatchSize int
	Custom1      string
	Custom2      string
	Custom3      string
}

type Message struct {
	ClientMsgId string      // The internal ID from the database
	APIMsgID    string      // The ID assigned by the SMS provider in the send response
	Destination []string    // List of mobile numbers to send to
	Text        string      // The message to send
	From        string      // Optional, provide a description of the sender
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
func SendSMS(msg, eml string, ns []string) error {
	log.Printf("User %v sending message '%v' to %v recipients.", eml, msg, len(ns))

	if !Config.SMSEnabled {
		return errors.New("SendSMS disabled in config, not sending")
	}

	err := splitBatchAndSend(msg, eml, ns) // Split message into batches if required by provider

	return err
}

// GetNumberStatus retrieves the delivery status of the last-sent message to a specific MSISDN
func GetNumberStatus(n string) (string, error) {
	mID, sendLogID, st, err := DB.getLastSMSID(n)
	if err != nil {
		return "", err
	}

	if st != "0" {
		return st, nil
	}

	stDesc, err := getStatus(mID, sendLogID)
	return stDesc, err
}

func getStatus(apiID, sendLogID string) (string, error) {
	resp := callMethod(Message{APIMsgID: apiID}, Config.SMSProvider.Name+"GetStatus").(SMSResponse)
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

func splitBatchAndSend(msg, eml string, ns []string) error {
	var err error
	bs := Config.SMSProvider.MaxBatchSize
	if bs > 0 && len(ns) > bs {
		err = sendSMSBatch(msg, eml, ns[:bs])
		splitBatchAndSend(msg, eml, ns[bs:])
	} else {
		err = sendSMSBatch(msg, eml, ns)
	}
	return err
}

func sendSMSBatch(msg, eml string, ns []string) error {

	m := Message{
		Destination: ns,
		Text:        msg,
		//ClientMsgId: "0",
		From:     "IMQS",
		Provider: Config.SMSProvider,
	}
	resp := callMethod(m, Config.SMSProvider.Name+"SendSMS").(SMSResponse)

	// Run the DB entry code in a go function to prevent delays in responding
	// to the front-end while performing large insert DB operations.
	go func() {
		if err := DB.createSMSData(msg, eml, &resp); err != nil {
			errors.New("SendSMS DB error")
		}
	}()

	return resp.Error
}
