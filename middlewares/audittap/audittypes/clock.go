package audittypes

import "time"

// Clock provides injectable time (supports testing)
type Clock interface {
	Now() time.Time
}

type normalClock struct{}

func (c normalClock) Now() time.Time {
	return time.Now()
}

// TheClock is a clock that is replaceable during testing.
var TheClock Clock = normalClock{}
