# Go Programming Guide

## Introduction

Go is an open source programming language supported by Google. It is simple, reliable, and efficient.

## Key Features

- **Fast compilation**: Go compiles very quickly.
- **Garbage collection**: Automatic memory management.
- **Built-in concurrency**: Goroutines and channels make concurrency easy.
- **Rich standard library**: Go comes with a comprehensive standard library.

## Hello World

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
```

## Variables

Go has various ways to declare variables:

```go
var name string = "Go"
age := 10 // short declaration
const pi = 3.14 // constant
```

## Functions

Functions in Go can return multiple values:

```go
func add(a, b int) (int, error) {
    if a < 0 || b < 0 {
        return 0, fmt.Errorf("negative numbers not allowed")
    }
    return a + b, nil
}
```

## Concurrency

Use goroutines for concurrent programming:

```go
func main() {
    ch := make(chan int)
    go func() {
        ch <- 42
    }()
    result := <-ch
    fmt.Println(result)
}
```
