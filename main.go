package main

import (
	"fmt"
	"log"
	"main/act"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	// Create a new registry for policies and actions with a default policy.
	reg := act.NewRegistry("passthrough")

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
			result, ok := data[act.Verdict].(bool)
			if ok {
				fmt.Printf("Log: %v\n", result)
				return true, nil
			}
			return false, fmt.Errorf("Invalid data type")
		},
	}
	reg.Actions["call"] = act.Action{
		Name: "call",
		Sync: false,
		Run: func(data map[string]any) (bool, error) {
			result, ok := data[act.Verdict].(bool)
			if ok {
				fmt.Printf("Call: %v\n", result)
				return true, nil
			}
			return false, fmt.Errorf("Invalid data type")
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

	// Print the verdict.
	for _, v := range verdict {
		if v != nil {
			data := struct {
				MatchedPolicy string
				Sync          bool
				Verdict       bool
				Metadata      map[string]any
			}{
				MatchedPolicy: v.Data[act.MatchedPolicy].(string),
				Sync:          v.Data[act.Sync].(bool),
				Verdict:       v.Data[act.Verdict].(bool),
				Metadata:      v.Data[act.Metadata].(map[string]any),
			}
			// fmt.Printf("Data: %v\n", data)
			fmt.Printf("Matched: %v\n", data.MatchedPolicy)
			fmt.Printf("Sync: %v\n", data.Sync)
			fmt.Printf("Verdict: %v\n", data.Verdict)
			fmt.Printf("Metadata: %v\n", data.Metadata)
			result, err := reg.Actions[data.MatchedPolicy].Run(v.Data)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Result: %v\n", result)
			}
		} else {
			fmt.Println("No verdict")
		}
		fmt.Println("-----------------------------")
	}
}
