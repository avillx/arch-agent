package task

import "time"

// type Reglament interface {
// 	NextTime() time.Duration
// }

type At struct {
	time time.Time
}

func ExecuteAt(time time.Time) *At {
	return &At{
		time: time,
	}
}

func (a *At) NextTime() time.Duration {
	return time.Until(a.time)
}
