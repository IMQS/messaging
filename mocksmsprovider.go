package messaging

import (
	"log"
	"math/rand"
	"strconv"
	"time"
)

// MockProviderSendSMS simulates a SMS provider for testing purposes
// The messages always succeed with this provider.
func (m message) MockProviderSendSMS() SMSResponse {
	log.Println("Simulating sending message with MockProviderSendSMS")
	var msRess []SendSMSResponseMessage
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)
	for i := range m.Destination {
		msRes := SendSMSResponseMessage{
			To:        m.Destination[i],
			MessageID: strconv.Itoa(random.Intn(100000000)),
			ErrorCode: "0",
			ErrorDesc: "",
		}
		msRess = append(msRess, msRes)
	}
	sr := SMSResponse{
		Error: nil,
		Data:  msRess,
	}

	return sr
}

// MockProviderGetStatus simulates a SMS provider for testing purposes
func (m message) MockProviderGetStatus() SMSResponse {
	log.Println("Simulating getting status with MockProviderGetStatus")

	// Randomly succeed, fail or delay messages
	var errC string
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)
	rReturn := random.Intn(10)
	switch {
	case rReturn < 1:
		errC = "405" // Mock failed
	case rReturn >= 2 && rReturn <= 8:
		errC = "101" // Mock success
	default:
		errC = "056" // Mock sent (in progress)
	}
	var msRess []SendSMSResponseMessage
	msRess = append(msRess, SendSMSResponseMessage{
		MessageID: m.ProviderID,
		ErrorCode: mockProviderMapCode(errC),
		ErrorDesc: mockProviderMapCode(errC),
		Quantity:  1,
	})

	sr := SMSResponse{
		Error: nil,
		Data:  msRess,
	}

	return sr
}

func mockProviderMapCode(c string) string {
	switch {
	case c == "101":
		return "delivered"
	case c == "405":
		return "failed"
	default:
		return "sent"
	}
}
