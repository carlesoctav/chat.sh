# chat.sh, Copilot-chat Command line interface (CLI) to make your copilot looks like chatGPT

why do we need antoher CLI when we already have the gh Copilot CLI?
1. It's not just CLI questions; it can be used for other purposes as well.
2. I want to learn how to build CLI using go.
3. It supports saving your chat with the -chat-file flag and provides more context from your clipboard with the -clipboard-context flag.


## Installation
You can fin dthe binary file on the release page. Alternatively, if you have Go, you can install it using `go install cmd/chatsh.go`.


## Usage
```cmd
Usage: chatsh [COMMAND]
    Examples:
    chatsh chat "How to install Git on Windows"
    Chat with Copilot. This command will prompt text to the Copilot API.

    chatsh chat -chat-file ./test.json "Rewrite everything with Go"
    Same as above, but with previous chat context specified by ./test.json.

    chatsh chat -h

    chatsh setup
    Setup Copilot for this CLI.

    Available Commands:
    setup   Setup OAuth
    chat    Chat with Copilot
```

```cmd
Usage: chatsh chat [-chat-file] [-prompt] [-clipboard-context] [query]
  -chat-file string (file)
        add previous chat context specified by chat-file
  -clipboard-context
        add clipboardas another context
  -prompt string
        custom prompt
```

It also supports stdin, so you can do something like this:
```cmd
cat apple.py | chatsh chat "$(cat -) what's this code doing"
```








