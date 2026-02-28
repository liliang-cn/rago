// Package main shows how to use the rago agent library
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	http.DefaultTransport.(*http.Transport).ForceAttemptHTTP2 = true

	ctx := context.Background()

	fmt.Println("Creating agent...")
	svc, err := agent.NewBuilder("assistant").
		Build()
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer svc.Close()
	fmt.Println("Agent created successfully")

	fmt.Println("Planning...")
	plan, err := svc.Plan(ctx, "写一个 Go 语言的 Hello World 程序")
	if err != nil {
		log.Fatalf("Plan failed: %v", err)
	}
	fmt.Printf("Plan ID: %s\n", plan.ID)

	fmt.Println("Executing...")
	result, err := svc.Execute(ctx, plan.ID)
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}
	fmt.Printf("Result:\n%v\n", result.FinalResult)

	svc.SaveToFile(fmt.Sprintf("%v", result.FinalResult), "./hello.go")
	fmt.Println("Saved to ./hello.go")
}
