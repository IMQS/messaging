package messaging

import (
	"errors"
	"log"
)

// CR: This is quite confusing, because it looks like a 'message' can represent a message
// both in the unsent and the sent phase. Perhaps it would be better if there was just
// a 'message' which was raw immutable input data, and then the 'response' data structure
// would hold all of the results from the actual message sender.
// It's much clearer if you can write "response = send(message)",
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
// CR: I'm not sure that there should be an "Error" value here. Any such error should probably
// just be a function return value.
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

	// CR: Random string literals should be constants
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
// CR: Is this comment accurate? Surely what we should be doing is checking on
// the status of all messages with unknown status, which were went within the
// last X seconds, where X is some reasonable constant, like 30 minutes.
// We should not be building that logic on an "interval". Just use straight
// time difference.
// "i" is a really bad name for a function parameter, but this parameter shouldn't
// even exist.
func UpdateStatus(i int) {
	aIDs, err := DB.getUnresolvedIDs(i)

	if err != nil {
		log.Printf("UpdateStatus failed: %v", err)
	}
	for x := 0; x < len(aIDs); x++ {
		getStatus(aIDs[x][0], aIDs[x][1])
	}

}

// CR: I'm not convinced that we should use a recursive function here, because
// it's harder for most people to understand. If we were all coding in pure functional
// languages, then it probably wouldn't be an issue, but generally that's not the
// case here, so it's extra mental effort to understand the behaviour of this function.
// One particular thing of note here, is that we are throwing away error messages
// from recursive invocations.
// If that is intentional, then I'd say that's just plain wrong - we should never throw
// away errors.
// However, I wouldn't be surprised if this was an unintentional bug, which adds
// credibility to my claim that recursive functions are harder to get right.
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
		From:        "IMQS",
		Provider:    Config.SMSProvider,
	}
	resp := callMethod(m, Config.SMSProvider.Name+"SendSMS").(SMSResponse)

	sendID, err := DB.createSMSData(msg, eml, &resp)
	if err != nil {
		return "", errors.New("SendSMS DB error")
	}

	return sendID, resp.Error
}
