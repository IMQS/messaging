package messaging

import (
	"log"
	"time"
)

// This may not work well when the user submits a large amount of messages
// just before the start of the new interval.  The messages would then not
// have enough time to progress through the networks and will likely not
// have a final state (i.e. delivered or failed) yet.  We should look at
// starting a new timer after every batch send, ensuring that at least the
// configured amount of time will pass before checking for the delivery status.
func startInterval() {
	if Config.DeliveryStatus.Enabled {
		log.Printf("Starting ticker to check delivery status every %v minutes", Config.DeliveryStatus.UpdateInverval)
		ticker := time.NewTicker(time.Minute * time.Duration(Config.DeliveryStatus.UpdateInverval))
		quit := make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					UpdateStatus(Config.DeliveryStatus.UpdateInverval)
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()
	}
}
