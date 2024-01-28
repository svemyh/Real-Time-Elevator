package timer

import (
	"time"
)

var timer *time.Timer
var timerActive bool

func timer_start(duration float64) {
	timer = time.NewTimer(time.Duration(duration) * time.Second)
	timerActive = true
}

func timer_stop() {
	if timer != nil {
		timer.Stop()
	}
	timerActive = false
}

func timer_timedOut() bool {
	if !timerActive {
		return false
	}
	select {
	case <-timer.C:
		timerActive = false
		return true
	default:
		return false
	}
}
