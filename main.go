package main

import (
	"fmt"
	"log"
	"main/act"
	"os"
	"sync"

	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
)

func main() {
	// Create a new registry for policies and actions with a default policy.
	reg := act.NewRegistry("passthrough")
	defer reg.Queue.Release()

	// Read policies from YAML file.
	yamlFile, err := os.ReadFile("policies.yaml")
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}

	var data map[string]interface{}
	if err = yaml.Unmarshal(yamlFile, &data); err != nil {
		log.Fatalf("Error unmarshalling YAML: %v", err)
	}

	// Register policies.
	if v, ok := data["policies"]; ok {
		if policies, ok := v.([]interface{}); ok {
			for _, p := range policies {
				if p, ok := p.(map[string]interface{}); ok {
					name := p["name"].(string)
					policy := p["policy"].(string)
					metadata, ok := p[act.Metadata].(map[string]interface{})
					if !ok {
						metadata = map[string]interface{}{}
					}

					p := act.NewPolicy(name, policy, metadata)
					if p != nil {
						reg.Policies[name] = *p
					}
				}
			}
		}
	}

	// Register actions.
	reg.Actions["terminate"] = act.Action{
		Name: "terminate",
		Sync: true,
		Run: func(data map[string]any) (bool, error) {
			result, ok := data[act.Verdict].(bool)
			if ok {
				return result, nil
			}
			return result, fmt.Errorf("Invalid data type")
		},
	}
	reg.Actions["log"] = act.Action{
		Name: "log",
		Sync: false,
		Run: func(data map[string]any) (bool, error) {
			log.Printf("%v", data)
			return true, nil
		},
	}
	reg.Actions["call"] = act.Action{
		Name: "call",
		Sync: false,
		Run: func(data map[string]any) (bool, error) {
			log.Printf("%v", data)
			return true, nil
		},
	}

	// Apply a signal to the registry.
	verdict := reg.Apply([]act.Signal{
		// {
		// 	Name:     "passthrough",
		// 	Metadata: map[string]any{},
		// 	Sync:     true,
		// },
		{
			Name: "terminate",
			Metadata: map[string]any{
				"terminate": true,
				"hook": map[string]any{
					"name": "OnTrafficFromClient",
				},
			},
			Sync: true,
		},
		{
			Name: "log",
			Metadata: map[string]any{
				"log":   true,
				"msg":   "Hello, world!",
				"level": "info",
			},
			Sync: false,
		},
		{
			Name: "call",
			Metadata: map[string]any{
				"call":   true,
				"method": "POST",
				"url":    "https://example.com",
				"body":   "Hello, world!",
				"headers": map[string]string{
					"Content-Type": "text/plain",
				},
			},
			Sync: false,
		},
	})

	var wg sync.WaitGroup
	wg.Add(2)

	go func(r *act.Registry, wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			select {
			case i := <-r.Inputs:
				matchedPolicy := cast.ToString(i.Data[act.MatchedPolicy])
				fmt.Println(reg.Actions[matchedPolicy].Run(i.Data))
			}
		}
	}(reg, &wg)

	go func(r *act.Registry, wg *sync.WaitGroup) {
		defer wg.Done()
		// Print the verdict.
		for _, v := range verdict {
			if v != nil {
				data := struct {
					MatchedPolicy string
					Sync          bool
					Verdict       bool
					Metadata      map[string]any
				}{
					MatchedPolicy: cast.ToString(v.Data[act.MatchedPolicy]),
					Sync:          cast.ToBool(v.Data[act.Sync]),
					Verdict:       cast.ToBool(v.Data[act.Verdict]),
					Metadata:      cast.ToStringMap(v.Data[act.Metadata]),
				}
				// fmt.Printf("Data: %v\n", data)
				fmt.Printf("Matched: %v\n", data.MatchedPolicy)
				fmt.Printf("Sync: %v\n", data.Sync)
				fmt.Printf("Verdict: %v\n", data.Verdict)
				fmt.Printf("Metadata: %v\n", data.Metadata)
				if data.Sync {
					result, err := reg.Actions[data.MatchedPolicy].Run(v.Data)
					if err != nil {
						fmt.Printf("Error: %v\n", err)
					} else {
						fmt.Printf("Result: %v\n", result)
					}
				} else {
					go func(r *act.Registry, res *act.Result) {
						err := r.Queue.Queue(res)
						if err != nil {
							log.Println(err)
						}
					}(r, v)
				}
			} else {
				fmt.Println("No verdict")
			}
			fmt.Println("-----------------------------")
		}
	}(reg, &wg)

	wg.Wait()
}
