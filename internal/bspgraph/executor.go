package bspgraph

import "context"

// ExecutorCallbacks encapsulates a series of callbacks that are invoked by an
// Executor instance on a graph. All callbacks are optional and will be ignored
// if not specified.
type ExecutorCallbacks struct {
	// PreStep, if defined, is invoked before running the next superstep.
	// This is a good place to initialize variables, aggregators etc. that
	// will be used for the next superstep.
	PreStep func(ctx context.Context, g *Graph) error

	// PostStep, if defined, is invoked after running a superstep.
	PostStep func(ctx context.Context, g *Graph, activeInStep int) error

	// ShouldPostStepKeepRunning, if defined, is invoked after running a superstep
	// to decide whether the stop condition for terminating the run has
	// been met. The number of the active vertices in the last step is
	// passed as the second argument.
	ShouldPostStepKeepRunning func(ctx context.Context, g *Graph, activeInStep int) (bool, error)
}

func patchEmptyCallbacks(cb *ExecutorCallbacks) {
	if cb.PreStep == nil {
		cb.PreStep = func(context.Context, *Graph) error { return nil }
	}

	if cb.PostStep == nil {
		cb.PostStep = func(context.Context, *Graph, int) error { return nil }
	}

	if cb.ShouldPostStepKeepRunning == nil {
		cb.ShouldPostStepKeepRunning = func(context.Context, *Graph, int) (bool, error) { return true, nil }
	}
}

// ExecutorFactory is a function that creates new Executor instances.
type ExecutorFactory func(*Graph, ExecutorCallbacks) *Executor

// Executor wraps a Graph instance and provides an orchestration layer for
// executing super-steps until an error occurs or an exit condition is met.
// Users can provide an optional set of callbacks to be executed before and
// after each super-step.
type Executor struct {
	g   *Graph // g value must be a pointer of Graph type.
	cbs ExecutorCallbacks // cbs can be a value or an expression that returns type ExecutorCallbacks.
}

// NewExecutor returns an Executor instance for graph g that invokes the
// provided list of callbacks inside each execution loop.
func NewExecutor(g *Graph, cbs ExecutorCallbacks) *Executor {
	patchEmptyCallbacks(&cbs)
	g.superstep = 0

	return &Executor{
		g:   g,
		cbs: cbs,
	}
}

// RunToCompletion keeps executing supersteps until the context expires, an
// error occurs or one of the Pre and or ShouldPostStepKeepRunning callbacks specified at
// configuration time returns false.
func (ex *Executor) RunToCompletion(ctx context.Context) error {
	return ex.run(ctx, -1)
}

// RunSteps executes at most numOfSteps supersteps unless the context expires, an
// error occurs or one of the Pre and or ShouldPostStepKeepRunning callbacks specified at
// configuration time returns false.
func (ex *Executor) RunSteps(ctx context.Context, numOfSteps int) error {
	return ex.run(ctx, numOfSteps)
}

// Graph returns the graph instance associated with this executor.
func (ex *Executor) Graph() *Graph {
	return ex.g
}

// Superstep returns the current graph superstep.
func (ex *Executor) Superstep() int {
	return ex.g.Superstep()
}

func (ex *Executor) run(ctx context.Context, maxSteps int) error {
	var (
		activeInStep int
		err          error
		keepRunning  bool
		cbs          = ex.cbs
	)

	for ; maxSteps != 0; ex.g.superstep, maxSteps = ex.g.superstep+1, maxSteps-1 {
		if err = ensureContextNotExpired(ctx); err != nil {
			break
		} else if err = cbs.PreStep(ctx, ex.g); err != nil {
			break
		} else if activeInStep, err = ex.g.step(); err != nil {
			break
		} else if err = cbs.PostStep(ctx, ex.g, activeInStep); err != nil {
			break
		} else if keepRunning, err = cbs.ShouldPostStepKeepRunning(ctx, ex.g, activeInStep); !keepRunning || err != nil {
			break
		}
	}

	return err
}

func ensureContextNotExpired(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
