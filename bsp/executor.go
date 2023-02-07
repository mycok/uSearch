package bsp

import "context"

// ExecutorCallbacks encapsulates a series of callbacks that are invoked by an
// Executor instance on a graph. All callbacks are optional and will be ignored
// if not specified.
type ExecutorCallbacks struct {
	// PreStep, if defined, is invoked before executing a super step. This is a
	// good place to initialize variables, aggregators etc. that are needed by
	// the super step.
	PreStep func(ctx context.Context, g *Graph) error

	// PostStep, if defined, is invoked after running a super step.
	PostStep func(ctx context.Context, g *Graph, activeInStep int) error

	// ShouldRunAnotherStep if defined, is invoked after running a super step.
	// it checks whether the condition for terminating the entire run has
	// been met and if so the executor terminates, else the executor will
	// execute another step.
	ShouldRunAnotherStep func(
		ctx context.Context, g *Graph, activeInStep int,
	) (bool, error)
}

func initWithDefaultCallbacks(cb *ExecutorCallbacks) {
	if cb.PreStep == nil {
		cb.PreStep = func(ctx context.Context, g *Graph) error {
			return nil
		}
	}

	if cb.PostStep == nil {
		cb.PostStep = func(ctx context.Context, g *Graph, activeInStep int) error {
			return nil
		}

	}

	if cb.ShouldRunAnotherStep == nil {
		cb.ShouldRunAnotherStep = func(
			ctx context.Context, g *Graph, activeInStep int,
		) (bool, error) {

			return true, nil
		}
	}
}

// ExecutorFactory is a function that creates new Executor instances.
// Note: Should be used for cases where lazy object creation is desired.
type ExecutorFactory func(g *Graph, cb ExecutorCallbacks) *Executor

// Executor serves as an orchestration layer for execution of super steps until
// an error occurs or an exit condition is met.
// Clients can provide an optional set of callbacks to be executed before and
// after each super-step.
type Executor struct {
	g   *Graph
	cbs ExecutorCallbacks
}

// NewExecutor initializes and returns an Executor instance.
func NewExecutor(g *Graph, cbs ExecutorCallbacks) *Executor {
	initWithDefaultCallbacks(&cbs)
	g.superStep = 0

	return &Executor{
		g:   g,
		cbs: cbs,
	}
}

// Graph returns the graph instance associated with this executor.
func (ex *Executor) Graph() *Graph {
	return ex.g
}

// SuperStep returns the current graph super step.
func (ex *Executor) SuperStep() int {
	return ex.g.SuperStep()
}

// RunToCompletion runs the super step until either the context expires, an
// error occurs or one of the Pre and or ShouldRunAnotherStep callbacks returns
// false.
func (ex *Executor) RunToCompletion(ctx context.Context) error {
	return ex.run(ctx, -1)
}

// RunSteps executes at most numOfSteps super steps unless the context expires, an
// error occurs or one of the Pre and or ShouldRunAnotherStep callbacks returns
// false.
func (ex *Executor) RunSteps(ctx context.Context, numOfSteps int) error {
	return ex.run(ctx, numOfSteps)
}

func (ex *Executor) run(ctx context.Context, maxSteps int) error {
	var (
		activeInStep int
		err          error
		shouldRun    bool
		cbs          = ex.cbs
	)

	for ; maxSteps != 0; ex.g.superStep, maxSteps = ex.g.superStep+1, maxSteps-1 {
		if err = ensureContextNotExpired(ctx); err != nil {
			break
		} else if err = cbs.PreStep(ctx, ex.g); err != nil {
			break
		} else if activeInStep, err = ex.g.step(); err != nil {
			break
		} else if err = cbs.PostStep(ctx, ex.g, activeInStep); err != nil {
			break
		} else if shouldRun, err = cbs.ShouldRunAnotherStep(
			ctx, ex.g,
			activeInStep,
		); !shouldRun || err != nil {
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
