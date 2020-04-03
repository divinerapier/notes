# Observer

## Code

``` go
package main

import (
	"fmt"
	"time"
)

func main() {
	timer := NewClockTimer()
	NewDigitalClock(1, timer)
	timer.Tick()
}

type Subject interface {
	Attach(Observer) error
	Detach(Observer)
	Notify()
}

type Observer interface {
	Update(Subject)
	ID() int64
}

type BaseSubject struct {
	m map[int64]Observer
}

func (s *BaseSubject) Attach(o Observer) error {
	id := o.ID()
	if _, exists := s.m[id]; exists {
		return fmt.Errorf("exists observer: %d", id)
	}
	s.m[id] = o
	return nil
}

func (s *BaseSubject) Detach(o Observer) {
	id := o.ID()
	delete(s.m, id)
}

func (s *BaseSubject) Notify() {
	for k := range s.m {
		o := s.m[k]
		o.Update(s)
	}
}

type ClockTimer struct {
	BaseSubject
}

func (ct *ClockTimer) Tick() {
	ct.Notify()
}

func NewClockTimer() *ClockTimer {
	return &ClockTimer{
		BaseSubject: BaseSubject{
			m: make(map[int64]Observer),
		},
	}
}

type DigitalClock struct {
	s  Subject
	id int64
}

func (dc *DigitalClock) Update(s Subject) {
	fmt.Printf("%d: %s digital clock\n", dc.id, time.Now().String())
}

func (dc *DigitalClock) ID() int64 {
	return dc.id
}

func NewDigitalClock(id int64, s Subject) *DigitalClock {
	dc := &DigitalClock{
		s:  s,
		id: id,
	}
	dc.s.Attach(dc)
	return dc
}
```
