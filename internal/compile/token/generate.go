//go:build generate

//go:generate go run ../../../tools/generate_tokens.go tokens.json generated.go
package generate
