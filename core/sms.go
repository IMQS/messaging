package messaging

import "errors"

const (
	Delivered = "delivered"
	Failed    = "failed"
	Sent      = "sent"
)

type SMSSender interface {
	SendSMS(s *MessagingServer, m message) ([]SendSMSResponseMessage, error)
	GetStatus(s *MessagingServer, m message) ([]SendSMSResponseMessage, error)
}

// The message struct is used to define a new SMS message that needs to be sent
// to a list of mobile numbers (Destination).
type message struct {
	ID          string            // The internal ID from the database
	ProviderID  string            // The ID assigned by the SMS provider in the send response
	Destination []string          // List of string mobile numbers to send to
	Text        string            // The text message to send
	From        string            // Optional, provide a description for the sender
	Provider    ConfigSmsProvider // SMS Provider to send the messages through
}

// SendSMSResponseMessage struct represents the response of a message contained
// within a "send" API call.
type SendSMSResponseMessage struct {
	To        string
	MessageID string
	ErrorCode string
	ErrorDesc string
	Segments  int
}

func (s *MessagingServer) getSender(n string) SMSSender {
	m := map[string]SMSSender{
		"Clickatell":   ClickatellSender{},
		"MockProvider": MockProviderSender{},
	}
	return m[n]
}

// SendSMSMessages implements REST APIs for SMS providers, as configured in the config.
// It also stores all messages in a DB for later reference
func (s *MessagingServer) SendSMSMessages(msg, eml string, ns []string) (string, error) {
	s.Log.Debugf("User %v sending message '%v' to %v recipients.", eml, msg, len(ns))

	if !s.Config.SMSProvider.Enabled {
		return "", errors.New("SendSMS disabled in config, not sending")
	}

	sendID, err := splitBatchAndSend(msg, eml, ns, s) // Split message into batches if required by provider

	return sendID, err
}

// GetNumberStatus retrieves the delivery status of the last-sent message to a specific MSISDN
func (s *MessagingServer) GetNumberStatus(n string) (string, error) {
	mID, sendLogID, st, err := s.DB.getLastSMSID(n)
	if err != nil {
		return "", err
	}

	if st != Sent {
		return st, nil
	}

	stDesc, err := getStatus(mID, sendLogID, s)
	return stDesc, err
}

func getStatus(apiID, sendLogID string, s *MessagingServer) (string, error) {
	m := message{ProviderID: apiID}
	smsSender := s.getSender(s.Config.SMSProvider.Name)
	resp, err := smsSender.GetStatus(s, m)

	if err != nil {
		return "", err
	}
	stCode := resp[0].ErrorCode

	if err := s.DB.updateSMSData(apiID, stCode, resp[0].ErrorDesc, sendLogID, resp[0].Segments); err != nil {
		return "", errors.New("SendSMS DB error")
	}

	return stCode, nil
}

// UpdateStatus is executed on an interval, finding all unresolved delivery
// statuses from the 30 minutes and retrieving it from the service provider
func UpdateStatus(s *MessagingServer) {
	aIDs, err := s.DB.getUnresolvedIDs("30m")

	if err != nil {
		s.Log.Errorf("UpdateStatus failed: %v", err)
	}
	for x := 0; x < len(aIDs); x++ {
		getStatus(aIDs[x][0], aIDs[x][1], s)
	}

}

func splitBatchAndSend(msg, eml string, ns []string, s *MessagingServer) (string, error) {
	var err error
	var sendID string

	bs := s.Config.SMSProvider.MaxBatchSize
	ratio := float32(len(ns)) / float32(bs)

	// BUG(dbf): We are losing the full result set and only returning the final sendID and err
	for ratio > 0 {
		if ratio > 1 {
			sendID, err = sendSMSBatch(msg, eml, ns[:bs], s)
			ns = ns[bs:]
			ratio = float32(len(ns)) / float32(bs)
		} else {
			sendID, err = sendSMSBatch(msg, eml, ns, s)
			return sendID, err
		}
	}

	return sendID, err
}

func sendSMSBatch(msg, eml string, ns []string, s *MessagingServer) (string, error) {
	m := message{
		Destination: ns,
		Text:        msg,
		From:        "IMQS",
		Provider:    s.Config.SMSProvider,
	}
	smsSender := s.getSender(s.Config.SMSProvider.Name)
	resp, sendErr := smsSender.SendSMS(s, m)

	sendID, err := s.DB.createSMSData(msg, eml, resp, sendErr)
	if err != nil {
		return "", errors.New("SendSMS DB error")
	}

	return sendID, sendErr
}

func allowOnlyASCII(str string) string {
	b := make([]byte, len(str))
	var bl int
	for i := 0; i < len(str); i++ {
		c := str[i]
		if c >= 32 && c < 127 {
			b[bl] = c
			bl++
		}
	}
	return string(b[:bl])
}
