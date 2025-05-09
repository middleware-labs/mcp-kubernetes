package main

// func main() {
// 	// Create a simple JSON-RPC request
// 	request := map[string]interface{}{
// 		"jsonrpc": "2.0",
// 		"id":      1,
// 		"method":  "callTool", // Try with the simpler method name
// 		"params": map[string]interface{}{
// 			"name": "Run-kubectl-command",
// 			"arguments": map[string]interface{}{
// 				"command": "version",
// 			},
// 		},
// 	}

// 	// Marshal the request to JSON
// 	jsonData, err := json.Marshal(request)
// 	if err != nil {
// 		fmt.Printf("Error marshaling JSON: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Start the server command
// 	cmd := exec.Command("../mcp-kubernetes")
// 	cmd.Stderr = os.Stderr

// 	// Create pipes for stdin and stdout
// 	stdin, err := cmd.StdinPipe()
// 	if err != nil {
// 		fmt.Printf("Error creating stdin pipe: %v\n", err)
// 		os.Exit(1)
// 	}

// 	stdout, err := cmd.StdoutPipe()
// 	if err != nil {
// 		fmt.Printf("Error creating stdout pipe: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Start the command
// 	if err := cmd.Start(); err != nil {
// 		fmt.Printf("Error starting command: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Send the JSON-RPC request
// 	fmt.Println("Sending request:", string(jsonData))
// 	stdin.Write(jsonData)
// 	stdin.Write([]byte("\n"))

// 	// Read the response
// 	buf := make([]byte, 4096)
// 	n, err := stdout.Read(buf)
// 	if err != nil {
// 		fmt.Printf("Error reading response: %v\n", err)
// 	} else {
// 		fmt.Println("Response:", string(buf[:n]))

// 		// Try to pretty-print the JSON response
// 		var prettyJSON bytes.Buffer
// 		json.Indent(&prettyJSON, buf[:n], "", "  ")
// 		fmt.Println("\nPretty Response:")
// 		fmt.Println(prettyJSON.String())
// 	}

// 	// Clean up
// 	stdin.Close()
// 	cmd.Wait()
// }
