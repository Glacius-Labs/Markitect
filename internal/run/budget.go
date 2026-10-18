package run

import (
	"math/big"
	"time"
)

// Capacity is finite assignment capacity, not money, tokens or elapsed delivery
// time. Reserve.Runs is always zero. Policy is a Host projection of already
// decoded, source-bound configuration; run does not parse execution YAML.
type Capacity struct {
	Runs          int64
	SessionStarts int64
	Steps         int64
	ExecutionTime time.Duration
}
type BudgetPolicy struct {
	Ceilings             Capacity
	Assignment           Capacity
	Reserve              Capacity
	MaxStepTimeout       time.Duration
	HostOperationTimeout time.Duration
}

func (p BudgetPolicy) validate(reviews int64) error {
	return p.validateDemand(reviewDemand{Subjects: reviews, Starts: reviews, Steps: reviews})
}

func (p BudgetPolicy) validateDemand(demand reviewDemand) error {
	for _, c := range []Capacity{p.Ceilings, p.Assignment} {
		if c.Runs <= 0 || c.SessionStarts <= 0 || c.Steps <= 0 || c.ExecutionTime <= 0 {
			return ErrInvalid
		}
	}
	if !fits(p.Assignment, p.Ceilings) || p.Reserve.Runs != 0 || p.Reserve.SessionStarts < demand.Starts || p.Reserve.Steps < demand.Steps || p.Reserve.ExecutionTime <= 0 || !fits(p.Reserve, p.Assignment) || p.MaxStepTimeout <= 0 || p.HostOperationTimeout <= 0 {
		return ErrInvalid
	}
	return nil
}
func fits(a, b Capacity) bool {
	return a.Runs <= b.Runs && a.SessionStarts <= b.SessionStarts && a.Steps <= b.Steps && a.ExecutionTime <= b.ExecutionTime
}

type Purpose string

const (
	Writing    Purpose = "writing"
	Assessment Purpose = "assessment"
	// Preparation is an assessment that prepares writing, such as the
	// expectations step before a work step; like writing it leaves the
	// completion reserve untouched (DEC-049).
	Preparation Purpose = "preparation"
)

type InvocationKind string

const (
	SessionStart InvocationKind = "session_start"
	AgentStep    InvocationKind = "agent_step"
	ProjectCheck InvocationKind = "project_check"
	ToolProcess  InvocationKind = "tool_process"
)

type Reservation struct {
	ID      string
	Kind    InvocationKind
	Purpose Purpose
	Maximum time.Duration
}

// valid checks intrinsic record shape independently of the current balance and
// selected timeout. The codec and admission use this same closed value contract.
func (r Reservation) valid() bool {
	return identifier(r.ID) && (r.Purpose == Writing || r.Purpose == Assessment || r.Purpose == Preparation) && r.Maximum > 0 &&
		(r.Kind == SessionStart || r.Kind == AgentStep || r.Kind == ProjectCheck || r.Kind == ToolProcess) &&
		(r.Kind != ProjectCheck || r.Purpose == Assessment)
}

type InvocationState string

const (
	Reserved  InvocationState = "reserved"
	Uncertain InvocationState = "uncertain"
	Settled   InvocationState = "settled"
	Released  InvocationState = "not_dispatched"
)

type Invocation struct {
	Reservation Reservation
	CapRevision uint64
	State       InvocationState
	Elapsed     time.Duration
	Observation string
}

// Consumption preserves exact accumulated nanoseconds, even after multiple
// reported overruns exceed time.Duration's range. Remaining is never negative.
type Consumption struct {
	Runs, SessionStarts, Steps int64
	ExecutionNanoseconds       string
}
type BudgetView struct {
	Consumed  Consumption
	Held      Capacity
	Remaining Capacity
	Overrun   bool
}

func budget(p BudgetPolicy, runs int64, invocations map[string]Invocation) BudgetView {
	v := BudgetView{Consumed: Consumption{Runs: runs}, Remaining: p.Assignment}
	used := new(big.Int)
	for _, invocation := range invocations {
		r := invocation.Reservation
		if invocation.State == Released {
			continue
		}
		starts, steps := int64(0), int64(0)
		if r.Kind == SessionStart {
			starts = 1
		}
		if r.Kind == AgentStep {
			steps = 1
		}
		if invocation.State == Settled {
			v.Consumed.SessionStarts += starts
			v.Consumed.Steps += steps
			used.Add(used, big.NewInt(int64(invocation.Elapsed)))
			v.Overrun = v.Overrun || invocation.Elapsed > r.Maximum
		} else {
			v.Held.SessionStarts += starts
			v.Held.Steps += steps
			v.Held.ExecutionTime += r.Maximum
		}
	}
	v.Consumed.ExecutionNanoseconds = used.String()
	v.Remaining.Runs -= runs
	v.Remaining.SessionStarts -= v.Consumed.SessionStarts + v.Held.SessionStarts
	v.Remaining.Steps -= v.Consumed.Steps + v.Held.Steps
	left := new(big.Int).Sub(big.NewInt(int64(p.Assignment.ExecutionTime)), used)
	left.Sub(left, big.NewInt(int64(v.Held.ExecutionTime)))
	if left.Sign() < 0 {
		v.Remaining.ExecutionTime = 0
	} else {
		v.Remaining.ExecutionTime = time.Duration(left.Int64())
	}
	if v.Overrun {
		v.Remaining = Capacity{}
	}
	return v
}
func cost(r Reservation) Capacity {
	c := Capacity{ExecutionTime: r.Maximum}
	if r.Kind == SessionStart {
		c.SessionStarts = 1
	}
	if r.Kind == AgentStep {
		c.Steps = 1
	}
	return c
}
func admit(p BudgetPolicy, view BudgetView, r Reservation) error {
	if !r.valid() || r.Kind == AgentStep && r.Maximum > p.MaxStepTimeout {
		return ErrInvalid
	}
	c := cost(r)
	if view.Overrun || !fits(c, view.Remaining) {
		return ErrBudget
	}
	if r.Purpose == Writing || r.Purpose == Preparation {
		left := view.Remaining
		left.SessionStarts -= c.SessionStarts
		left.Steps -= c.Steps
		left.ExecutionTime -= c.ExecutionTime
		if !fits(p.Reserve, left) {
			return ErrBudget
		}
	}
	return nil
}
