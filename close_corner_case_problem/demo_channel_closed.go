package main

import (
	"context"
	"errors"
	"log"
	"time"
	//	mgo "github.com/globalsign/mgo"
)

// Graceful represents an interface to the shutdown method
type Graceful interface {
	shutdown(ctx context.Context /*, session *mgo.Session,*/, closedChannel chan bool)
}

type graceful struct{}

func (t graceful) shutdown(ctx context.Context /*session *mgo.Session,*/, closedChannel chan bool) {
	//session.Close()

	log.Printf("in shutdown()")
	time.Sleep(time.Second) // simulate a session.Close() taking a while
	log.Printf("shutdown: sending to closedChannel")

	defer func() { // remove this defer and you will see a "panic: send on closed channel"
		if x := recover(); x != nil {
			log.Printf("recovered a panic: %v: ", x)
			// do nothing ... just handle timing corner case and avoid "panic: send on closed channel"
		}
	}()

	log.Printf("about to send 'true' over closedChannel")
	closedChannel <- true
	log.Printf("sent 'true' over closedChannel") // !!! we dont't see this !!! due to panic()
	return
}

var (
	start    Graceful = graceful{}
	timeLeft          = 1000 * time.Millisecond
)

// Close represents mongo session closing within the context deadline
func Close(ctx context.Context /*, session *mgo.Session*/) error {
	closedChannel := make(chan bool)
	defer func() {
		log.Printf("doing Close() defer")
		close(closedChannel)
	}()

	// Make a copy of timeLeft so that we don't modify the global var
	closeTimeLeft := timeLeft
	if deadline, ok := ctx.Deadline(); ok {
		// Add some time to timeLeft so case where ctx.Done in select
		// statement below gets called before time.After(timeLeft) gets called.
		// This is so the context error is returned over hardcoded error.
		closeTimeLeft = deadline.Sub(time.Now()) + (10 * time.Millisecond)
	}

	go func() {
		start.shutdown(ctx /*, session,*/, closedChannel)
		log.Printf("returned from shutdown")
		return
	}()

	log.Printf("waiting on select ...")
	select {
	case <-time.After(closeTimeLeft):
		log.Printf("timed out")
		return errors.New("closing mongo timed out")
	case <-closedChannel:
		log.Printf("received channelClosed")
		return nil
	case <-ctx.Done():
		log.Printf("got ctx.Done()")
		return ctx.Err()
	}
}

// Demonstrate a corner case of delays that when the added "defer func()" in shutdown() is commented
// out will cause a panic()
// NOTE: you need to comment it out to see the problem, that is the adding of the recover() code
//       fixes this corner case.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeLeft = 800 * time.Millisecond
	err := Close(ctx /*, session.Copy()*/)

	time.Sleep(100 * time.Millisecond)

	log.Printf("main(): err is: %v", err)

	time.Sleep(1000 * time.Millisecond) // simulate enough further delay before application shuts down for the delay in shutdown() to expire
}
