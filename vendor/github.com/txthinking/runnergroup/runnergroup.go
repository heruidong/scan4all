package runnergroup

import (
	"sync"
	"time"
)

// RunnerGroup is like sync.WaitGroup,
// the diffrence is if one task stops, all will be stopped.
type RunnerGroup struct {
	Runners []*Runner
	End     chan byte
}

type Runner struct {
	// Start is a blocking function.
	Start func() error
	// Stop is not a blocking function, if Stop called, must let Start return.
	// Notice: Stop maybe called multi times even if before start.
	Stop   func() error
	lock   sync.Mutex
	status int
}

func New() *RunnerGroup {
	g := &RunnerGroup{}
	g.Runners = make([]*Runner, 0)
	g.End = make(chan byte)
	return g
}

func (g *RunnerGroup) Add(r *Runner) {
	g.Runners = append(g.Runners, r)
}

// Call Wait after all task have been added,
// Return the first ended start's result.
func (g *RunnerGroup) Wait() error {
	e := make(chan error)
	for _, v := range g.Runners {
		v.status = 1
		go func(v *Runner) {
			err := v.Start()
			v.lock.Lock()
			v.status = 0
			v.lock.Unlock()
			select {
			case <-g.End:
			case e <- err:
			}
		}(v)
	}
	err := <-e
	for _, v := range g.Runners {
		for {
			v.lock.Lock()
			if v.status == 0 {
				v.lock.Unlock()
				break
			}
			v.lock.Unlock()
			_ = v.Stop()
			time.Sleep(300 * time.Millisecond)
		}
	}
	close(g.End)
	return err
}

// Call Done if you want to stop all.
// return the stop's return which is not nil, do not guarantee,
// because starts may ended caused by itself.
func (g *RunnerGroup) Done() error {
	if len(g.Runners) == 0 {
		return nil
	}
	var e error
	for _, v := range g.Runners {
		for {
			v.lock.Lock()
			if v.status == 0 {
				v.lock.Unlock()
				break
			}
			v.lock.Unlock()
			if err := v.Stop(); err != nil {
				if e == nil {
					e = err
				}
			}
			time.Sleep(300 * time.Millisecond)
		}
	}
	<-g.End
	return e
}
