package messaging

import (
	"math/rand"
	"strconv"
	"time"
)

type MockProviderSender struct {
}

// SendSMS simulates a SMS provider for testing purposes
// The messages always succeed with this provider.
func (p MockProviderSender) SendSMS(s *MessagingServer, m message) ([]SendSMSResponseMessage, error) {
	s.Log.Info("Simulating sending message with MockProviderSendSMS\n")
	var msRess []SendSMSResponseMessage
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)
	for _, dm := range m.Destination {
		msRes := SendSMSResponseMessage{
			To:        dm,
			MessageID: strconv.Itoa(random.Intn(100000000)),
			ErrorCode: "0",
			ErrorDesc: "",
			Segments:  1,
		}
		msRess = append(msRess, msRes)
	}
	return msRess, nil
}

// GetStatus simulates a SMS provider for testing purposes
func (p MockProviderSender) GetStatus(s *MessagingServer, m message) ([]SendSMSResponseMessage, error) {
	s.Log.Info("Simulating getting status with MockProviderSender GetStatus\n")

	// Randomly succeed, fail or delay messages
	var errC string
	time.Sleep(1 * time.Second)
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)
	rReturn := random.Intn(10)
	switch {
	case rReturn < 1:
		errC = "405" // Mock failed
	case rReturn >= 8:
		errC = "056" // Mock sent (in progress)
	default:
		errC = "101" // Mock success
	}
	var msRess []SendSMSResponseMessage
	msRess = append(msRess, SendSMSResponseMessage{
		MessageID: m.ProviderID,
		ErrorCode: mapCode(errC),
		ErrorDesc: mapCode(errC),
		Segments:  1,
	})

	return msRess, nil
}

func mapCode(c string) string {
	m := map[string]string{
		"101": Delivered,
		"405": Failed,
		"056": Sent,
	}
	return m[c]
}
