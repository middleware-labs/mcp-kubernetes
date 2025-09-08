package kubectl

import (
	"fmt"
	"sync"
	"time"
)

type PendingRequest struct {
	mu     sync.Mutex
	done   bool
	result string
	err    error
	cond   *sync.Cond
}

func NewPendingRequest() *PendingRequest {
	pr := &PendingRequest{}
	pr.cond = sync.NewCond(&pr.mu)
	return pr
}

func (pr *PendingRequest) Wait(timeout time.Duration) (string, error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if !pr.done {
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		doneCh := make(chan struct{})
		go func() {
			pr.mu.Lock()
			for !pr.done {
				pr.cond.Wait()
			}
			pr.mu.Unlock()
			close(doneCh)
		}()

		select {
		case <-doneCh:
			// finished
		case <-timer.C:
			return "", fmt.Errorf("timeout waiting for response")
		}
	}
	return pr.result, pr.err
}

func (pr *PendingRequest) Complete(result string, err error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	if pr.done {
		return
	}
	pr.done = true
	pr.result = result
	pr.err = err
	pr.cond.Broadcast()
}
