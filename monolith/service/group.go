package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
)

// Service describes a service for the Links 'R' Us monolithic application.
type Service interface {
	// Name returns the name of the service.
	Name() string

	// Run executes the service and blocks until the context gets cancelled
	// or an error occurs.
	Run(context.Context) error
}

// Group is a list of Service instances that can execute in parallel.
type Group []Service


// Execute executes all Service instances in the group using the provided context.
// Calls to Run block until all services have completed executing either because
// the context was cancelled or any of the services reported an error.
func (g Group) Execute(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	executionCtx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	var wg sync.WaitGroup
	wg.Add(len(g))
	errChan := make(chan error, len(g))

	for _, s := range g {
		go func(s Service) {
			defer wg.Done()

			if err := s.Run(executionCtx); err != nil {
				errChan <- fmt.Errorf("%s: %w", s.Name(), err)

				cancelFn()
			}

		}(s)
	}

	// Keep running until the execution context gets cancelled, then wait for
	// all spawned service go-routines to exit.
	<-executionCtx.Done()

	wg.Wait()

	// Collect and accumulate any reported errors.
	var err error
	close(errChan)

	for srvErr := range errChan {
		err = multierror.Append(err, srvErr)
	}

	return err
}