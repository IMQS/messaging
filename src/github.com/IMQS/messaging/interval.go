package messaging

import (
	"log"
	"time"
)

func startInterval() {
	if Config.DeliveryStatus.Enabled {
		log.Printf("Starting interval timer to run every %v minutes", Config.DeliveryStatus.UpdateInverval)
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
