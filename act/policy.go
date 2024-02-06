package act

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/spf13/cast"
)

const (
	Verdict       = "verdict"
	Decision      = "decision"
	Sync          = "sync"
	ActionSync    = "actionSync"
	SignalSync    = "signalSync"
	MatchedPolicy = "matched"
	Metadata      = "metadata"
)

type Result struct {
	Data map[string]any
}

func (s *Result) Bytes() []byte {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return b
}

var DefaultResult = func() *Result {
	return &Result{
		Data: map[string]any{
			MatchedPolicy: "passthrough",
			Verdict:       true, // TODO: Check the policy and return the result.
			Sync:          true,
		},
	}
}

type Policy struct {
	program *vm.Program
	opts    []expr.Option

	Name     string
	Policy   string
	Metadata map[string]any
	Sync     bool
}

func NewPolicy(name string, policy string, args map[string]any, opts ...expr.Option) *Policy {
	print := expr.Function("print", func(a ...any) (any, error) {
		return fmt.Println(a)
	})
	opts = append(opts, expr.Env(NewInput(name, args, map[string]any{}, true, true)))
	opts = append(opts, print)

	program, err := expr.Compile(policy, opts...)
	if err != nil {
		// TODO: Log or return error.
		log.Println(err)
		return nil
	}

	return &Policy{
		program:  program,
		opts:     opts,
		Name:     name,
		Policy:   policy,
		Metadata: args,
	}
}

func (p *Policy) String() string {
	return p.Name
}

func (p *Policy) Eval(data Input) *Result {
	// TODO: Merge the outputs of all the policies and return the result.

	// // Run previous policies.
	// for _, prev := range p.prev {
	// 	_, err := expr.Run(prev.program, expr.Env(data))
	// 	if err != nil {
	// 		// TODO: Log or return error.
	// 		return nil
	// 	}
	// }

	// Check if the action and signal sync are in sync and warn if not.
	if data.ActionSync != data.SignalSync {
		slog.LogAttrs(
			context.TODO(), slog.LevelWarn, "Action and signal sync mismatch", slog.Attr{
				Key:   "action",
				Value: slog.StringValue(data.Name),
			}, slog.Attr{
				Key:   "sync",
				Value: slog.BoolValue(cast.ToBool(data.Signal[Sync])),
			},
		)
	}

	// Run the policy.
	output, err := expr.Run(p.program, data)
	if err != nil {
		// TODO: Log or return error.
		slog.LogAttrs(context.TODO(), slog.LevelError, "Error running policy", slog.Attr{
			Key:   "error",
			Value: slog.StringValue(err.Error()),
		})
		return nil
	}
	o, ok := output.(bool)
	if !ok {
		// TODO: Log or return error.
		return nil
	}

	// // Run next policies.
	// for _, next := range p.next {
	// 	_, err := expr.Run(next.program, expr.Env(data))
	// 	if err != nil {
	// 		// TODO: Log or return error.
	// 		return nil
	// 	}
	// }

	return &Result{
		Data: map[string]any{
			MatchedPolicy: p.Name,
			Verdict:       o,
			ActionSync:    data.ActionSync,
			SignalSync:    data.SignalSync,
			Metadata:      data.Signal,
		},
	}
}
