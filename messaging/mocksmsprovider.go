package messaging

import (
	"log"
	"math/rand"
	"strconv"
	"time"
)

// MockProviderSendSMS simulates a SMS provider for testing purposes
// The messages always succeed with this provider.
func (m Message) MockProviderSendSMS() SMSResponse {
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
func (m Message) MockProviderGetStatus() SMSResponse {
	log.Println("Simulating getting status with MockProviderGetStatus")

	// Randomly succeed or fail messages
	var errC string
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)
	if random.Intn(10) >= 3 {
		errC = "004"
	} else {
		errC = "007"
	}
	var msRess []SendSMSResponseMessage
	msRess = append(msRess, SendSMSResponseMessage{
		MessageID: m.APIMsgID,
		ErrorCode: errC,
		ErrorDesc: "Received",
		Quantity:  1,
	})

	sr := SMSResponse{
		Error: nil,
		Data:  msRess,
	}

	return sr
}
