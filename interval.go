package messaging

import "time"

type IntervalService struct {
	quit chan int
}

func (is *IntervalService) Stop() {
	is.quit <- 0
}

func (s *MessagingServer) startInterval() {
	if s.Config.DeliveryStatus.Enabled {
		s.Log.Infof("Starting ticker to check delivery status every %v", s.Config.DeliveryStatus.UpdateInterval)
		d, err := time.ParseDuration(s.Config.DeliveryStatus.UpdateInterval)
		if err != nil {
			s.Log.Warnf("Could not start ticker due to invalid time configuration: %v", err.Error())
			return
		}
		ticker := time.NewTicker(d)
		s.Interval.quit = make(chan int)
		go func() {
			for {
				select {
				case <-ticker.C:
					UpdateStatus(s)
				case <-s.Interval.quit:
					ticker.Stop()
					return
				}
			}
		}()
	}
}
