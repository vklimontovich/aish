# 🤖 aish - AI Shell Assistant

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> Your AI-powered command line assistant that turns natural language into shell commands!

Example: 
```
❯ ./aish **what is the largest process in memory right now and what is the full path to a command that started it**
💬 Tool wants to run: ps aux --sort=-%mem | awk 'NR==2{print $11, $6}'
Run this command? [y/N] y
ℹ️ Running command: ps aux --sort=-%mem | awk 'NR==2{print $11, $6}'
ps: illegal option -- -
usage: ps [-AaCcEefhjlMmrSTvwXx] [-O fmt | -o fmt] [-G gid[,gid...]]
          [-u]
          [-p pid[,pid...]] [-t tty[,tty...]] [-U user[,user...]]
       ps [-L]

💬 Tool wants to run: ps aux | sort -nrk 4 | head -n 1 | awk '{print $11, $6}'
Run this command? [y/N] y
ℹ️ Running command: ps aux | sort -nrk 4 | head -n 1 | awk '{print $11, $6}'
/Users/vklmn/Applications/IntelliJ 3596096

💬 Tool wants to run: ps aux | sort -nrk 4 | head -n 1 | awk '{print $11, $6}'
Run this command? [y/N] y
ℹ️ Running command: ps aux | sort -nrk 4 | head -n 1 | awk '{print $11, $6}'
/Users/vklmn/Applications/IntelliJ 3596128
```

## ✨ Features

- 🔮 Convert natural language instructions into shell commands
- 🔄 Interractive mode if LLM needs any clarifications
- 🔑 Simple configuration management
- 🚀 Support for OpenAI, Helicone API. Other backends support is coming

## 🚀 Installation

### 🍺 Homebrew (macOS and Linux)

```bash
brew tap vklimontovich/aish
brew install aish
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
