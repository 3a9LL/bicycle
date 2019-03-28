package spidy

import "time"
type Limiter struct {
	reqChanC	chan chan struct{}

	rate		int
	takenCount	int
}

func NewLimiter(rate int) *Limiter {
	l := &Limiter{}
	l.reqChanC		= make(chan chan struct{}, 1024)
	l.takenCount	= 0
	l.rate			= rate

	go l.Service()
	return l
}

func (l *Limiter) Service() {
	c := <-l.reqChanC
	ticker := time.NewTicker(1 * time.Second)
	l.takenCount = 1
	c <- struct{}{}
	for {
		select {
		case c := <-l.reqChanC:
			if l.takenCount >= l.rate{
				<-ticker.C
				l.takenCount = 0
			}
			l.takenCount++
			c <- struct{}{}
		case <-ticker.C:
			l.takenCount = 0
		}
	}
}

func (l *Limiter) Take(){
	c := make(chan struct{}, 1)
	l.reqChanC <- c
	<-c
}