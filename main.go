package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
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
			metadata := cast.ToStringMapString(data[act.Metadata])
			attrs := make([]slog.Attr, 0, len(metadata))
			for k, v := range metadata {
				attrs = append(attrs, slog.Attr{
					Key:   k,
					Value: slog.StringValue(v),
				})
			}
			slog.LogAttrs(
				context.TODO(), slog.LevelInfo, "Logger called", attrs...,
			)
			return true, nil
		},
	}
	reg.Actions["call"] = act.Action{
		Name: "call",
		Sync: false,
		Run: func(data map[string]any) (bool, error) {
			metadata := cast.ToStringMapString(data[act.Metadata])
			attrs := make([]slog.Attr, 0, len(metadata))
			for k, v := range metadata {
				attrs = append(attrs, slog.Attr{
					Key:   k,
					Value: slog.StringValue(v),
				})
			}
			slog.LogAttrs(
				context.TODO(), slog.LevelInfo, "Logger called", attrs...,
			)
			return true, nil
		},
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
				result, err := reg.Actions[matchedPolicy].Run(i.Data)
				attrs := []slog.Attr{}
				attrs = append(attrs, slog.Attr{
					Key:   "result",
					Value: slog.BoolValue(result),
				})
				attrs = append(attrs, slog.Attr{
					Key:   "name",
					Value: slog.StringValue(matchedPolicy),
				})
				if err != nil {
					attrs = append(attrs, slog.Attr{
						Key:   "error",
						Value: slog.StringValue(fmt.Sprintf("%v", err)),
					})
				}
				slog.LogAttrs(
					context.TODO(), slog.LevelInfo, "Result of running action", attrs...,
				)
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
					ActionSync    bool
					SignalSync    bool
					Verdict       bool
					Metadata      map[string]any
				}{
					MatchedPolicy: cast.ToString(v.Data[act.MatchedPolicy]),
					ActionSync:    cast.ToBool(v.Data[act.ActionSync]),
					SignalSync:    cast.ToBool(v.Data[act.SignalSync]),
					Verdict:       cast.ToBool(v.Data[act.Verdict]),
					Metadata:      cast.ToStringMap(v.Data[act.Metadata]),
				}

				slog.LogAttrs(
					context.TODO(), slog.LevelInfo, "Verdict", slog.Attr{
						Key:   "matched",
						Value: slog.StringValue(data.MatchedPolicy),
					}, slog.Attr{
						Key:   "sync",
						Value: slog.BoolValue(data.ActionSync),
					}, slog.Attr{
						Key:   "verdict",
						Value: slog.BoolValue(data.Verdict),
					},
				)

				if data.ActionSync {
					result, err := reg.Actions[data.MatchedPolicy].Run(v.Data)
					attrs := []slog.Attr{}
					attrs = append(attrs, slog.Attr{
						Key:   "result",
						Value: slog.BoolValue(result),
					})
					attrs = append(attrs, slog.Attr{
						Key:   "name",
						Value: slog.StringValue(data.MatchedPolicy),
					})
					if err != nil {
						attrs = append(attrs, slog.Attr{
							Key:   "error",
							Value: slog.StringValue(fmt.Sprintf("%v", err)),
						})
					}
					slog.LogAttrs(
						context.TODO(), slog.LevelInfo, "Result of running action", attrs...,
					)
				} else {
					go func(r *act.Registry, res *act.Result) {
						err := r.Queue.Queue(res)
						if err != nil {
							log.Println(err)
						}
						slog.LogAttrs(
							context.TODO(), slog.LevelInfo, "Queued task", slog.Attr{
								Key:   "matched",
								Value: slog.StringValue(data.MatchedPolicy),
							},
						)
					}(r, v)
				}
			} else {
				slog.LogAttrs(context.TODO(), slog.LevelInfo, "No verdict")
			}
		}
	}(reg, &wg)

	wg.Wait()
}
