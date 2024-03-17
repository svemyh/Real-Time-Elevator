package timer

import (
	"time"
)

var timer *time.Timer
var timerActive bool

func DoorTimerStart(duration int64) {
	timer = time.NewTimer(time.Duration(duration) * time.Second)
	timerActive = true
}

func DoorTimerStop() {
	if timer != nil {
		timer.Stop()
	}
	timerActive = false
}

func DoorTimerTimedOut() bool {
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
