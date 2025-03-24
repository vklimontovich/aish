package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/pflag"
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
	configDir := filepath.Join(usr.HomeDir, ".aish")
	configPath = filepath.Join(configDir, "aish.json")

	cfg := Config{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			fmt.Printf("%s⚠️ Warning: Failed to parse config: %v\n", warnColor(""), err)
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
	fmt.Printf("%s✅ Config saved to %s\n", successColor(""), configPath)
}

// Exit with formatted error
func exitWithError(format string, a ...any) {
	fmt.Printf(errorColor("❌ ")+format+"\n", a...)
	os.Exit(1)
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
					"command": map[string]any{"type": "string", "description": "Shell command to execute. You can use pipes, && etc. The will be executed in the same shell as sh -c ..."},
					"final":   map[string]any{"type": "boolean", "description": "true if final command"},
				},
				"required": []string{"command", "final"},
			},
		},
	}}

	payload := map[string]any{
		"model":    "gpt-4o",
		"messages": messages,
		"tools":    toolDefs,
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
		fmt.Printf("%sℹ️ Payload:\n%s\n", infoColor(""), string(prettyJSON))
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
		fmt.Printf("%sℹ️ Response:\n%s\n", infoColor(""), string(prettyJSON))
	}
	var jsonResp map[string]any
	_ = json.Unmarshal(data, &jsonResp)
	return jsonResp
}

func main() {
	var useShell bool
	var shellArgs []string
	var helpFlag bool
	var versionFlag bool
	// Parse CLI flags
	pflag.StringArrayVar(&configArgs, "config", []string{}, "Set config value: --config ai_key=XXX")
	pflag.BoolVar(&useShell, "shell", false, "Run command directly, mainly for testing")
	pflag.BoolVarP(&unsafeFlag, "unsafe", "u", false, "Run without confirmation")
	pflag.BoolVar(&debugFlag, "debug", false, "Debug mode")
	pflag.BoolVarP(&helpFlag, "help", "h", false, "Display help information")
	pflag.BoolVarP(&versionFlag, "version", "v", false, "Display version information")
	pflag.Parse()
	args := pflag.Args()

	// Display version if requested
	if versionFlag {
		fmt.Printf("aish version %s\n", Version)
		os.Exit(0)
	}

	// Display help if requested
	if helpFlag {
		displayHelp()
		os.Exit(0)
	}

	cfg := loadConfig()

	if useShell {
		for i, arg := range os.Args {
			if arg == "--shell" {
				shellArgs = os.Args[i+1:]
				break
			}
		}
		cmdOutput, cmdError, resultCode := runShellCommand(strings.Join(shellArgs, " "))
		fmt.Println("stdout:", cmdOutput)
		fmt.Println("stderr:", cmdError)
		fmt.Println("result code:", resultCode)
		os.Exit(0)
	}

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
		exitWithError("OpenAPI key is not set. Please export AI_KEY, or run `aish --config ai_key=XXX`")
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
	You are an assistant that helps users with shell commands on %s. Your ONLY goal is to build a command that solves the user's problem eventually.
    If you need to ask for more information, use the run_shell_command function with final=false. The response of this auxiliary command will be provided to you in the next step.
	 
	🔹 You must ALWAYS respond with exactly ONE function call: run_shell_command.
	
	🔹 NEVER reply with plain text or multiple function calls. 
	🔹 NEVER skip the function call — use it even to ask for more info.
	
	Command Lifecycle:

	- If command is final, and gives the user a final result (it's important), call run_shell_command with final=true.
	- If more info is needed, you can request it by calling run_shell_command with final=false. The output of this command will be provided to you in the next step.
	
	Failure Handling:
	- If a final command fails, the error will be provided to you in the next step. Try to correct error and issue a new final command
	
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

		// Check if assistant responded with just content (no call)
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
		final := toolArgs["final"].(bool)

		if !unsafeFlag {
			confirmed, err := promptConfirm(cmd)
			if err != nil {
				exitWithError("Error confirming command: %v", err)
			}
			if !confirmed {
				fmt.Println(warnColor("ℹ️ Command aborted."))
				os.Exit(0)
			}
		}
		cmdOutput, cmdError, resultCode := runShellCommand(cmd)

		if final {
			if resultCode != 0 {
				fmt.Printf(errorColor("❌ Command failed with error: %s\n"), cmdError)
				os.Exit(resultCode)
			}
			fmt.Printf(successColor("✅ Command succeeded!\n"))
			os.Exit(0)
		}

		resultStr := "SUCCEDED"
		if resultCode != 10 {
			resultStr = "FAILED"
		}
		messages = append(messages,
			map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{toolCall}},
			map[string]any{"role": "tool", "tool_call_id": toolCall["id"], "content": fmt.Sprintf("The command %s. Stdout: %s. Stderr: %s.", resultStr, cmdOutput, cmdError)},
		)
	}
}

// Display help information about the tool
func displayHelp() {
	fmt.Printf("%s🚀 aish - AI Shell Assistant%s v%s\n\n", successColor(""), "", Version)

	fmt.Printf("%sDescription:%s\n", warnColor(""), "")
	fmt.Printf("  aish is a tool for people who love shell, but have bad memory.\n")
	fmt.Printf("  Simply describe what you want to do in plain English,\n")
	fmt.Printf("  and aish will suggest and execute the appropriate shell commands (it's safe, command will be confirmed before execution).\n\n")
	fmt.Printf("  Currently, aish works with OpenAI only and requires an API key.\n")

	fmt.Printf("%sUsage:%s\n", warnColor(""), "")
	fmt.Printf("  aish [options] \"your prompt here\"\n\n")

	fmt.Printf("%sSetup:%s\n", warnColor(""), "")
	fmt.Printf("  %s🔑 Configure your API key:%s\n", successColor(""), "")
	fmt.Printf("    aish --config ai_key=sk-your-openai-key\n\n")

	fmt.Printf("%sExample:%s\n", warnColor(""), "")
	fmt.Printf("  %s🔎 Find large files:%s\n", successColor(""), "")
	fmt.Printf("    aish \"find the 5 largest files in the current directory\"\n\n")

	fmt.Printf("  %s🔎 Find large files:%s\n", successColor(""), "")
	fmt.Printf("    aish \"what is the largest process in memory right now and what is the full path to a command that started it\"\n\n")

	fmt.Printf("%sOptions:%s\n", warnColor(""), "")
	fmt.Printf("  %s--config key=value%s   Set configuration (ai_key, helicone_key)\n", infoColor(""), "")
	fmt.Printf("  %s--unsafe, -u%s         Run commands without confirmation prompts\n", infoColor(""), "")
	fmt.Printf("  %s--debug%s              Enable debug mode to see API payloads and responses\n", infoColor(""), "")
	fmt.Printf("  %s--help, -h%s           Display this help information\n", infoColor(""), "")
	fmt.Printf("  %s--version, -v%s        Display version information\n\n", infoColor(""), "")
}
