package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/pflag"
)

// ANSI Colors
const (
	Red    = "\033[91m"
	Green  = "\033[92m"
	Yellow = "\033[93m"
	Cyan   = "\033[96m"
	Bold   = "\033[1m\033[4m"
	Reset  = "\033[0m"
)

// Config struct for ai_key and helicone_key
type Config struct {
	AIKey       string `json:"ai_key"`
	HeliconeKey string `json:"helicone_key"`
}

var (
	configPath  string
	configArgs  []string
	unsafeFlag  bool
	debugFlag   bool
	userPrompt  string
	apiKey      string
	heliconeKey string
	sessionID   string
)

// Load config from ~/.ai-cli/ai-cli.json and apply env overrides
func loadConfig() Config {
	usr, err := user.Current()
	if err != nil {
		exitWithError("Failed to get current user: %v", err)
	}
	configDir := filepath.Join(usr.HomeDir, ".ai-cli")
	configPath = filepath.Join(configDir, "ai-cli.json")

	cfg := Config{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			fmt.Printf("%s⚠️ Warning: Failed to parse config: %v%s\n", Yellow, err, Reset)
		}
	}

	// Env var override
	if val := os.Getenv("AI_KEY"); val != "" {
		cfg.AIKey = val
	}
	if val := os.Getenv("HELICONE_KEY"); val != "" {
		cfg.HeliconeKey = val
	}
	return cfg
}

// Save config to file
func saveConfig(cfg Config) {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		exitWithError("Failed to create config directory: %v", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		exitWithError("Failed to encode config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		exitWithError("Failed to write config: %v", err)
	}
	fmt.Printf("%s✅ Config saved to %s%s\n", Green, configPath, Reset)
}

// Exit with formatted error
func exitWithError(format string, a ...any) {
	fmt.Printf(Red+"❌ "+format+Reset+"\n", a...)
	os.Exit(1)
}

// Prompt user to confirm shell command
func promptConfirm(cmd string) bool {
	fmt.Printf("💬 Tool wants to run: %s%s%s\nRun this command? [y/N] ", Bold, cmd, Reset)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(line)) == "y"
}

// Run shell command, handle stop (final) vs intermediate
func runShellCommand(cmd string, stop bool) string {
	if !unsafeFlag && !promptConfirm(cmd) {
		fmt.Println(Yellow + "ℹ️ Command aborted." + Reset)
		os.Exit(0)
	}

	fmt.Printf("%sℹ️ Running command: %s%s\n", Cyan, cmd, Reset)
	execCmd := exec.Command("sh", "-c", cmd)
	output, err := execCmd.CombinedOutput()
	
	// Always display output
	fmt.Printf("%s\n", string(output))
	
	if err != nil {
		if stop {
			fmt.Printf(Red+"❌ Command failed with error%s\n", Reset)
			return "Command failed with error:\n" + string(output) + "\nPlease fix the command and try again."
		}
		fmt.Printf(Yellow+"⚠️ Command error%s\n", Reset)
		return "Error executing command:\n" + string(output)
	}
	if stop {
		fmt.Println(Green + "✅ Command executed." + Reset)
		os.Exit(0)
	}
	return string(output)
}

// Build and send API request, return parsed response
func sendAPIRequest(messages []map[string]any, cfg Config) map[string]any {
	toolDefs := []map[string]any{{
		"type": "function",
		"function": map[string]any{
			"name":        "run_shell_command",
			"description": "Execute a shell command.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{"type": "string", "description": "Shell command to execute"},
					"stop":    map[string]any{"type": "boolean", "description": "true if final command"},
				},
				"required": []string{"command", "stop"},
			},
		},
	}}

	payload := map[string]any{
		"model":       "gpt-4o",
		"messages":    messages,
		"tools":       toolDefs,
		"tool_choice": map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": "run_shell_command",
			},
		},
		"temperature": 0,
	}

	body, _ := json.Marshal(payload)
	if debugFlag {
		// Pretty-print the payload for debug output
		prettyJSON, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Printf(Cyan+"ℹ️ Payload:\n%s"+Reset+"\n", string(prettyJSON))
	}

	// Determine correct API host
	host := "api.openai.com"
	if cfg.HeliconeKey != "" {
		host = "oai.helicone.ai"
	}
	url := "https://" + host + "/v1/chat/completions"

	// Build HTTP request
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AIKey)
	if cfg.HeliconeKey != "" {
		req.Header.Set("Helicone-Auth", "Bearer "+cfg.HeliconeKey)
		req.Header.Set("Helicone-Session-Id", sessionID)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		exitWithError("API error: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		exitWithError("API error %d: %s\n%s", resp.StatusCode, resp.Status, data)
	}
	if debugFlag {
		// Pretty-print the response for debug output
		var prettyResp interface{}
		_ = json.Unmarshal(data, &prettyResp)
		prettyJSON, _ := json.MarshalIndent(prettyResp, "", "  ")
		fmt.Printf(Cyan+"ℹ️ Response:\n%s"+Reset+"\n", string(prettyJSON))
	}
	var jsonResp map[string]any
	_ = json.Unmarshal(data, &jsonResp)
	return jsonResp
}

func main() {
	// Parse CLI flags
	pflag.StringArrayVar(&configArgs, "config", []string{}, "Set config value: --config ai_key=XXX")
	pflag.BoolVarP(&unsafeFlag, "unsafe", "u", false, "Run without confirmation")
	pflag.BoolVar(&debugFlag, "debug", false, "Debug mode")
	pflag.Parse()
	args := pflag.Args()

	cfg := loadConfig()
	
	// Handle config commands
	if len(configArgs) > 0 {
		for _, configArg := range configArgs {
			// Check for key=value format
			parts := strings.SplitN(configArg, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				val := parts[1]
				
				switch key {
				case "ai_key":
					cfg.AIKey = val
				case "helicone_key":
					cfg.HeliconeKey = val
				default:
					exitWithError("Unknown config key: %s", key)
				}
				saveConfig(cfg)
				os.Exit(0)
			} else {
				exitWithError("Invalid config format. Use: --config ai_key=XXX")
			}
		}
	}

	if cfg.AIKey == "" {
		exitWithError("AI_KEY not set in config or environment")
	}
	apiKey = cfg.AIKey
	heliconeKey = cfg.HeliconeKey
	if heliconeKey != "" {
		sessionID = uuid.New().String()
	}

	userPrompt = strings.Join(args, " ")
	if strings.TrimSpace(userPrompt) == "" {
		exitWithError("No prompt provided.")
	}

	systemPrompt := fmt.Sprintf(`
	You are an assistant that helps users with shell commands on %s. Your ONLY goal is to build a command that solves the user’s problem.
	
	🔹 You must ALWAYS respond with exactly ONE function call: run_shell_command.
	
	🔹 NEVER reply with plain text or multiple function calls. 
	🔹 NEVER skip the function call — use it even to ask for more info.
	
	Command Lifecycle:
	- If ready to execute: call run_shell_command with stop=true.
	- If more info is needed or testing a partial command: use stop=false.
	- If a command with stop=false succeeds and looks final, REPEAT it with stop=true.
	
	Failure Handling:
	- If a final command fails, respond with a corrected command via run_shell_command.
	
	🔒 MANDATORY: EVERY reply = exactly ONE run_shell_command function call.
	`, runtime.GOOS)
	
	// Message history
	messages := []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": userPrompt},
	}

	for {
		resp := sendAPIRequest(messages, cfg)
		choice := resp["choices"].([]any)[0].(map[string]any)
		assistantMessage := choice["message"].(map[string]any)

		// Check if assistant responded with just content (no tool call)
		if content, hasContent := assistantMessage["content"].(string); hasContent && content != "" {
			fmt.Println(content)
			os.Exit(0)
		}

		toolCalls, ok := assistantMessage["tool_calls"].([]any)
		if !ok || len(toolCalls) == 0 {
			exitWithError("No tool calls in response. Check prompt or API config.")
		}

		// Process the tool call
		toolCall := toolCalls[0].(map[string]any)
		funcCall := toolCall["function"].(map[string]any)
		var toolArgs map[string]any
		if err := json.Unmarshal([]byte(funcCall["arguments"].(string)), &toolArgs); err != nil {
			exitWithError("Error parsing tool arguments: %v", err)
		}

		cmd := toolArgs["command"].(string)
		stop := toolArgs["stop"].(bool)
		
		// Execute the command and get the output string
		cmdOutput := runShellCommand(cmd, stop)
		
		// Add the assistant's message with tool_calls to the conversation history
		messages = append(messages,
			map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{toolCall}},
			// The tool response format: role=tool, tool_call_id=<id>, content=<string result>
			map[string]any{"role": "tool", "tool_call_id": toolCall["id"], "content": cmdOutput},
		)
	}
}