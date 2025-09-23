package manager

import (
	"os"
	"syscall"
	"testing"
	"time"
)

var WorkerID int

type Worker struct{}

// Example loop, it will be spwaned in a goroutine
func (w *Worker) Run(um UnitManager) {
	ticker := time.NewTicker(time.Second)

	// Worker's loop
	for {
		select {

		case <-ticker.C:
			// llog.Print("tick")

		// Read from channel if this worker unit should stop
		case <-um.ShouldStop():

			// Shutdown work for current unit
			w.Shutdown()

			// Notify manager that this unit is done.
			um.Done()
		}
	}
}

func (w *Worker) Shutdown() {
	// Do shutdown procedure for worker
}

func NewWorker() *Worker {
	return &Worker{}
}

func DoRun(pid chan int,
	quit chan<- bool,
	signals ...os.Signal) {

	pid <- os.Getpid()

	// Create a unit manager
	manager := NewManager()

	// Shutdown all units on SIGINT
	manager.ShutdownOn(signals...)

	// NewWorker returns a type implementing WorkUnit interface unit :=
	worker1 := NewWorker()
	worker2 := NewWorker()

	// Register the unit with the manager
	manager.AddUnit(worker1, "")
	manager.AddUnit(worker2, "")

	// Start the manager
	go manager.Start()

	// Wait for all units to shutdown gracefully through their `Shutdown` method
	quit <- <-manager.Quit

}

func TestRunMain(t *testing.T) {
	signals := map[string]os.Signal{
		"interrupt": os.Interrupt,
	}
	mainPid := make(chan int, 1)
	quit := make(chan bool)

	for name, sig := range signals {
		t.Logf("Testing signal: %s\n", name)

		go DoRun(mainPid, quit, sig)
		time.Sleep(3 * time.Second)
		syssig, ok := sig.(syscall.Signal)
		if !ok {
			t.Fatalf("Could not convert os.Signal to syscall.Signal")
		}
		syscall.Kill(<-mainPid, syssig)
		<-quit
	}
}
