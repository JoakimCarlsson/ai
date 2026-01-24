package main

import (
	"fmt"

	"github.com/joakimcarlsson/ai/prompt"
)

func main() {
	simpleExample()
	cachedExample()
	validationExample()
	customFuncsExample()
}

func simpleExample() {
	result, err := prompt.Process("Hello, {{.name}}!", map[string]any{
		"name": "World",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
}

func cachedExample() {
	cache := prompt.NewCache()

	tmpl, err := prompt.New("You are {{.role}}. Help with {{.task}}.",
		prompt.WithCache(cache),
		prompt.WithName("assistant"),
	)
	if err != nil {
		panic(err)
	}

	result, err := tmpl.Process(map[string]any{
		"role": "a coding assistant",
		"task": "debugging",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
}

func validationExample() {
	_, err := prompt.Process("Hello, {{.name}}!", map[string]any{},
		prompt.WithRequired("name"),
	)
	if err != nil {
		fmt.Println("Validation error:", err)
	}
}

func customFuncsExample() {
	result, err := prompt.Process(
		"{{upper .name}} - {{default \"anonymous\" .nickname}}",
		map[string]any{"name": "alice"},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
}
