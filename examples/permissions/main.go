package main

import (
	"context"
	"fmt"
	"log"
	"os"

	claudecode "github.com/ryanhill4L/agents-sdk"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func deleteFile(path string) error {
	return os.Remove(path)
}

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	readTool := tools.Function("readFile", "Read contents of a file", readFile)
	writeTool := tools.Function("writeFile", "Write content to a file", writeFile)
	deleteTool := tools.Function("deleteFile", "Delete a file", deleteFile)

	client, err := claudecode.NewClient(
		claudecode.WithAPIKey(apiKey),
		claudecode.WithModel("claude-3-5-sonnet-20241022"),
		claudecode.WithPermissionMode(types.PermissionDefault),
		claudecode.WithAllowedDirectories("./safe_dir"),
		claudecode.WithBlockedDirectories("/etc", "/usr", "/bin"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	client.RegisterTool(readTool)
	client.RegisterTool(writeTool)
	client.RegisterTool(deleteTool)

	permissionCallback := func(req types.PermissionRequest) types.PermissionResponse {
		fmt.Printf("\n[Permission Request]\n")
		fmt.Printf("Tool: %s\n", req.Tool)
		fmt.Printf("Operation: %s\n", req.Operation)
		fmt.Printf("Path: %s\n", req.Path)
		fmt.Print("Allow? (y/n): ")

		var response string
		fmt.Scanln(&response)

		if response == "y" || response == "yes" {
			return types.PermissionResponse{
				Allowed: true,
				Reason:  "User approved",
			}
		}

		return types.PermissionResponse{
			Allowed: false,
			Reason:  "User denied permission",
		}
	}

	client.SetPermissionCallback(permissionCallback)

	fmt.Println("File Operations with Permissions")
	fmt.Println("----------------------------------------")

	os.MkdirAll("./safe_dir", 0755)
	os.WriteFile("./safe_dir/test.txt", []byte("Hello, World!"), 0644)

	queries := []string{
		"Read the file ./safe_dir/test.txt",
		"Write 'Updated content' to ./safe_dir/new.txt",
		"Try to read /etc/passwd",
		"Delete the file ./safe_dir/test.txt",
	}

	ctx := context.Background()

	for _, query := range queries {
		fmt.Printf("\nYou: %s\n", query)

		response, err := client.SendMessage(ctx, query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Print("Claude: ")
		for _, block := range response.GetContent() {
			if textBlock, ok := block.(*types.TextBlock); ok {
				fmt.Println(textBlock.Text)
			}
		}
	}

	os.RemoveAll("./safe_dir")
}