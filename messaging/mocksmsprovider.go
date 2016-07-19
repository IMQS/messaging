package messaging

import (
	"log"
	"math/rand"
	"strconv"
	"time"
)

// TestMessagingSendSMS simulates a SMS provider for testing purposes
// The messages always succeed with this provider.
func (m Message) TestMessagingSendSMS() SMSResponse {
	log.Println("Simulating sending message with TestMessagingSendSMS")
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

// TestMessagingGetStatus simulates a SMS provider for testing purposes
func (m Message) TestMessagingGetStatus() SMSResponse {
	log.Println("Simulating getting status with TestMessagingGetStatus")

	var errC string
	if rand.Intn(10) >= 5 {
		errC = "004"
	} else {
		errC = "007"
	}
	var msRess []SendSMSResponseMessage
	msRess = append(msRess, SendSMSResponseMessage{
		MessageID: m.APIMsgID,
		ErrorCode: errC,
		ErrorDesc: "Received by recipient",
		Quantity:  1,
	})

	sr := SMSResponse{
		Error: nil,
		Data:  msRess,
	}

	return sr
}
