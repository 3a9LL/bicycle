package spidy

import "time"

type Limiter struct {
	reqChanC	chan chan int

	rate		int
	takenCount	int
}

func NewLimiter(rate int) *Limiter {
	l := &Limiter{}
	l.reqChanC		= make(chan chan int, 1024)
	l.takenCount	= 0
	l.rate			= rate

	go l.Service()
	return l
}

func (l *Limiter) Service() {
	c := <-l.reqChanC
	l.takenCount = 1
	c <- 1
	deadline := time.Now().Add(time.Second)
	for {
		select {
		case c := <-l.reqChanC:
			if l.takenCount >= l.rate{
				time.Sleep(deadline.Sub(time.Now()))
				l.takenCount = 0
				deadline = time.Now().Add(time.Second)
			}
			l.takenCount++
			c <- 1
		case <-time.After(deadline.Sub(time.Now())):
			l.takenCount = 0
			deadline = time.Now().Add(time.Second)
		}
	}
}

func (l *Limiter) Take(){
	c := make(chan int, 1)
	l.reqChanC <- c
	<-c
}
