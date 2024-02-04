package act

import (
	"log"
	"slices"
)

const defaultPolicy = "passthrough"

type Registry struct {
	Signals       map[string]Signal
	Actions       map[string]Action
	Policies      map[string]Policy
	DefaultPolicy string

	Positive []string
	Negative []string
}

type Input struct {
	Policy map[string]any `json:"policy"`
	Signal map[string]any `json:"signal"`
	Sync   bool
}

func NewInput(policy map[string]any, signal map[string]any, sync bool) Input {
	return Input{
		Policy: policy,
		Signal: signal,
		Sync:   sync,
	}
}

func NewRegistry(defaultPolicy string) *Registry {
	return &Registry{
		Signals: map[string]Signal{
			"passthrough": {
				Name: "passthrough",
				Metadata: map[string]any{
					Data: true,
				},
				Sync: true,
			},
		},
		Actions: map[string]Action{
			"passthrough": {
				Name: "passthrough",
				Metadata: map[string]any{
					Data: true,
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

	if action.Sync {
		return policy.Eval(NewInput(policy.Metadata, signal.Metadata, action.Sync))
	} else {
		return r.enqueue(policy, action, signal)
	}
}

func (r *Registry) enqueue(policy Policy, action Action, signal Signal) *Result {
	// TODO: Queue the action and return the result.
	/*go*/
	return policy.Eval(
		NewInput(policy.Metadata, signal.Metadata, action.Sync),
	)
	// return &Result{
	// 	Data: map[string]any{
	// 		Sync: false,
	// 	},
	// }
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
