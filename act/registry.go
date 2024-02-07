package act

import (
	"context"
	"encoding/json"
	"log"
	"slices"

	"github.com/golang-queue/queue"
	"github.com/golang-queue/queue/core"
)

const defaultPolicy = "passthrough"

type Registry struct {
	Signals       map[string]Signal
	Actions       map[string]Action
	Policies      map[string]Policy
	DefaultPolicy string

	Positive []string
	Negative []string

	// Job queue
	Results chan *Result
	Queue   *queue.Queue
}

type Input struct {
	Name       string
	Policy     map[string]any `json:"policy"`
	Signal     map[string]any `json:"signal"`
	ActionSync bool
	SignalSync bool
}

func NewInput(
	name string, policy map[string]any, signal map[string]any, actionSync, signalSync bool,
) Input {
	return Input{
		Name:       name,
		Policy:     policy,
		Signal:     signal,
		ActionSync: actionSync,
		SignalSync: signalSync,
	}
}

func NewRegistry(defaultPolicy string) *Registry {
	resultsCh := make(chan *Result, 2)

	return &Registry{
		Signals: map[string]Signal{
			"passthrough": {
				Name: "passthrough",
				Metadata: map[string]any{
					Verdict: true,
				},
				Sync: true,
			},
		},
		Actions: map[string]Action{
			"passthrough": {
				Name: "passthrough",
				Metadata: map[string]any{
					Verdict: true,
				},
				Sync: true,
				Run: func(data map[string]any) (bool, error) {
					return true, nil
				},
			},
		},
		Policies: map[string]Policy{
			"passthrough": {
				Name:   "passthrough",
				Policy: "true",
			},
		},
		DefaultPolicy: defaultPolicy,
		// TODO: Make this configurable, or use a better algorithm.
		Positive: []string{
			"passthrough", "forward", "accept", "allow", "permit", "pass",
		},
		Negative: []string{
			"terminate", "block", "drop", "reject", "reset",
		},
		Results: resultsCh,
		Queue: queue.NewPool(2,
			queue.WithFn(func(ctx context.Context, m core.QueuedMessage) error {
				v, ok := m.(*Result)
				if !ok {
					if err := json.Unmarshal(m.Bytes(), &v); err != nil {
						return err
					}
				}

				resultsCh <- v

				return nil
			}),
		),
	}
}

func (r *Registry) Apply(signals []Signal) []*Result {
	DefaultResults := []*Result{
		DefaultResult(),
	}

	if len(signals) == 0 {
		return DefaultResults
	}

	results := []*Result{}
	// Check for non-contradictory actions (forward vs. drop)
	hasConflict, positives, negatives := r.Conflict(signals)
	if hasConflict {
		log.Println("Conflicting signals, applying default policy.")
		log.Println("Positives:", positives)
		log.Println("Negatives:", negatives)
		// TODO: Let the user decide what to do in case of conflict.
		return DefaultResults
	}

	for _, signal := range signals {
		results = append(results, r.apply(signal))
	}

	if len(results) == 0 {
		return DefaultResults
	}

	return results
}

func (r *Registry) apply(signal Signal) *Result {
	action, ok := r.Actions[signal.Name]
	if !ok {
		return DefaultResult()
	}

	policy, ok := r.Policies[action.Name]
	if !ok {
		return DefaultResult()
	}

	// Action dictates the sync mode, not the signal.
	return policy.Eval(
		NewInput(signal.Name, policy.Metadata, signal.Metadata, action.Sync, signal.Sync))
}

func (r *Registry) Conflict(signals []Signal) (bool, []string, []string) {
	// TODO: Check for conflicting signals with a faster algorithm.
	positives := []string{}
	negatives := []string{}
	for _, signal := range signals {
		if slices.Contains(r.Positive, signal.Name) {
			positives = append(positives, signal.Name)
		} else if slices.Contains(r.Negative, signal.Name) {
			negatives = append(negatives, signal.Name)
		}
	}

	if len(positives) > 0 && len(negatives) > 0 {
		return true, positives, negatives
	}

	return false, positives, negatives
}
