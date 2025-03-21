# 🤖 aish - AI Shell Assistant

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> Your AI-powered command line assistant that turns natural language into shell commands!

## ✨ Features

- 🔮 Convert natural language instructions into shell commands
- 🛡️ Interactive confirmation before executing commands (unless in unsafe mode)
- 🔄 Command execution with real-time feedback
- 🔑 Simple configuration management
- 🚀 Support for OpenAI and Helicone API integration

## 🚀 Installation

### 🍺 Homebrew (macOS and Linux)

```bash
brew tap vklimontovich/aish
brew install aish
```

### 🔨 From Source

```bash
git clone https://github.com/vklimontovich/aish.git
cd aish
make build
```

## 🔧 Configuration

Set up your API keys (required before first use):

```bash
# Configure OpenAI API Key
ai --config ai_key=your_openai_api_key

# Optional: Configure Helicone API Key for request tracking
ai --config helicone_key=your_helicone_key
```

Alternatively, you can set environment variables:

```bash
export AI_KEY=your_openai_api_key
export HELICONE_KEY=your_helicone_key
```

## 📋 Usage

Simply describe what you want to do in natural language:

```bash
# Basic usage
ai find all PNG files larger than 1MB in the current directory

# List running processes sorted by memory usage
ai show me the top 5 processes using the most memory

# Get disk usage information in human-readable format
ai how much disk space am I using
```

### 🎛️ Command Line Options

- `--unsafe` or `-u`: Run commands without confirmation prompts
- `--debug`: Enable debug mode for verbose output
- `--config key=value`: Set configuration values

## 🧠 How It Works

Aish leverages OpenAI's API to interpret your natural language instructions and convert them into appropriate shell commands for your operating system. Before execution, it shows you the command and asks for confirmation (unless in unsafe mode).

## 🔒 Security Note

By default, aish always prompts for confirmation before executing any command. Use the `--unsafe` flag with caution, as it will execute commands without asking for confirmation.

## 📄 License

MIT License 