package act

import (
	"fmt"
	"log"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

const (
	Data          = "data"
	Decision      = "decision"
	Sync          = "sync"
	MatchedPolicy = "matched"
)

type Result struct {
	Data map[string]any
}

var DefaultResult = func() *Result {
	return &Result{
		Data: map[string]any{
			MatchedPolicy: "passthrough",
			Data:          true, // TODO: Check the policy and return the result.
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
}

func NewPolicy(name string, policy string, args map[string]any, opts ...expr.Option) *Policy {
	print := expr.Function("print", func(a ...any) (any, error) {
		return fmt.Println(a)
	})
	opts = append(opts, expr.Env(NewInput(args, map[string]any{}, true)))
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

	// Run the policy.
	output, err := expr.Run(p.program, data)
	if err != nil {
		// TODO: Log or return error.
		fmt.Println(err)
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
			Data:          o,
			Sync:          data.Sync,
		},
	}
}
