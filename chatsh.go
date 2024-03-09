package chatsh

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/atotto/clipboard"
	"github.com/google/uuid"
)

var (
    chatCmd = flag.NewFlagSet("chat", flag.ExitOnError)
    setupCmd = flag.NewFlagSet("setup", flag.ExitOnError)
)


var subcommands = map[string]*flag.FlagSet{
    chatCmd.Name(): chatCmd,
    setupCmd.Name(): setupCmd,
}

type Message struct {
    Role    string `json:"role,omitempty"`
    Content string  `json:"content"`
}

type ContentParser struct {
    Choices []struct {
        Message struct {
            Content string `json:"content"`
        } `json:"message"`
    } `json:"choices"`
}

type CopilotChatPayload struct {
    Messages    interface{} `json:"messages"`
    Model       string      `json:"model"`
    Temperature float64     `json:"temperature"`
    TopP        float64     `json:"top_p"`
    N           int64       `json:"n"`
    Stream      bool        `json:"stream"`
}

type CopilotSession struct{
    chat string
    githubToken string
    copilotToken string
    chatUrl string
    input io.Reader
    output io.Writer
    chatHistory []Message
    chatPrompt Message
    chatFile string
}

type copilotOption func(client *CopilotSession) error

func NewInputWithInputFromArgs (args []string) copilotOption{
    return func (c *CopilotSession) error {
        if len(args) < 1{
            return nil
        }

        c.chat = args[0]
        return nil
    }
}

func NewCopilotSession(opts ...copilotOption) (*CopilotSession, error){
    homeDir,  err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }

    filePath := filepath.Join(homeDir, ".config", ".chatsh")

    f, err := os.Open(filePath)

    if err != nil {
        return nil, err
    }

    var cache Cache

    err = json.NewDecoder(f).Decode(&cache)

    if err != nil {
        return nil, err
    }
    c := &CopilotSession{
        githubToken: cache.GithubToken,
        chat :"",
        input: os.Stdin,
        output: os.Stdout,
        chatHistory: []Message{},
        chatUrl: "https://api.githubcopilot.com/chat/completions",
        chatFile: "",
        chatPrompt: Message{
            Role: "system",
            Content: "\nYou are ChatGPT, a large language model trained by OpenAI.\nKnowledge cutoff: 2021-09\nCurrent model: gpt-4\n",
        },
    }
    for _, opt := range opts{
         err := opt(c)
        if err != nil {
            return nil, err
        }
    }
    return c, nil
}


func (c *CopilotSession) Authenticate() (error){ 
    url := "https://api.github.com/copilot_internal/v2/token"
    authenticateHeader := map[string]string{
        "authorization": fmt.Sprintf("token %s", c.githubToken),
        "editor-version": "vscode/1.80.1",
        "editor-plugin-version": "copilot-chat/0.4.1",
        "user-agent": "GitHubCopilotChat/0.4.1",
    }

    client := &http.Client{}
    req, err := http.NewRequest("GET", url, nil)

    if err != nil {
        return err
    }
    for k,v := range authenticateHeader {
        req.Header.Set(k,v)
    }

    res, err := client.Do(req)

    if err != nil {
        return err
    }

    defer res.Body.Close()
    var results struct{
        Token string `json:"token"`
    } 
    data, err := io.ReadAll(res.Body)

    if err != nil {
        return err
    }

    err = json.Unmarshal(data, &results)

    if err != nil {
        return err
    }  
    c.copilotToken = results.Token
    return nil
}


func (c *CopilotSession) Save() error {
    f, err := os.Create(c.chatFile)

    if err != nil {
        return err
    }
    defer f.Close()
    return json.NewEncoder(f).Encode(c.chatHistory)
}

func (c *CopilotSession) Chat() error{
    err := c.Authenticate()

    if err != nil {
        return err
    }

    headers := c.CreateHeader()

    messagePayload := []Message{}
    messagePayload = append(messagePayload, c.chatPrompt)

    for _, chatHis := range c.chatHistory{
        messagePayload= append(messagePayload, chatHis)
    }

    if c.chat == ""{
        scanner := bufio.NewScanner(c.input)
        text := ""
        for scanner.Scan() {
            text += scanner.Text()
        }

        messagePayload = append(messagePayload, Message{
            Role: "user",
            Content: text,
        })
    } else {
        messagePayload = append(messagePayload, Message{
            Role: "user",
            Content: c.chat,
        })
    }
    jsonBody := &CopilotChatPayload{
        Messages: messagePayload,
        Model:       "gpt-4",
        Temperature: 0.1,
        TopP:        1,
        N:           1,
        Stream:      false,
    }
    jsonData, err := json.Marshal(jsonBody)

    if err != nil {
        return err
    }

    req, _ := http.NewRequest("POST", c.chatUrl, bytes.NewReader(jsonData))
    for k, v := range headers {
        req.Header.Set(k, v)
    }

    client := &http.Client{}
    resp, err := client.Do(req)

    if err != nil {
        return err
    }

    defer resp.Body.Close()

    data, err := io.ReadAll(resp.Body)

    if err != nil {
        return err
    }

    var content ContentParser

    err = json.Unmarshal(data, &content)
    if err != nil {
        return err
    }

    if c.chatFile != ""{
        c.chatHistory = messagePayload[1:]
        c.chatHistory = append(c.chatHistory, Message{
            Role: "assistant",
            Content: content.Choices[0].Message.Content,
        })
        err = c.Save()
    }

    if err != nil {
        return err
    }

    fmt.Println(content.Choices[0].Message.Content)

    return nil
}

func (c *CopilotSession) SetChatFile(chatFile string) error {
    // i need to open file and populete the chatHistory :D
    f, err := os.Open(chatFile)
    c.chatFile = chatFile

    if errors.Is(err, fs.ErrNotExist){
        return nil
    }

    if err != nil {
        return err
    }
    defer f.Close()

    err = json.NewDecoder(f).Decode(&c.chatHistory)

    if err != nil {
        return err
    }
    
    return nil
}

func (c *CopilotSession) SetPrompt(prompt string) error {
    c.chatPrompt = Message{
        Role: "system",
        Content: prompt,
    }
    return nil
}


func (c *CopilotSession) CreateHeader ()map[string]string{
    return map[string]string{
        "Authorization":         "Bearer " + c.copilotToken,
        "X-Request-Id":          uuid.NewString(),
        "Editor-Version":        "vscode/1.83.1",
        "Editor-Plugin-Version": "copilot-chat/0.8.0",
        "Openai-Organization":   "github-copilot",
        "Openai-Intent":         "conversation-panel",
        "Content-Type":          "application/json; charset=utf-8",
        "User-Agent":            "GitHubCopilotChat/0.8.0",
        "Accept":                "*/*",
        "Accept-Encoding":       "gzip,deflate,br",
        "Connection":            "close",
    }
}


func MainAuthenticate() int {
    fmt.Println("get github token")
    err := githubAuthenticate()

    if err != nil {
        fmt.Fprintln(os.Stdout, err)
        return 0
    }
    fmt.Println("Authenticate Success")
    return 1

}

func MainChat(flag *flag.FlagSet) int {
    chatFile := flag.String("chat-file", "", "add previous chat context specified by chat-file")
    prompt := flag.String("prompt", "", "custom prompt")
    clipBoardContext := flag.Bool("clipboard-context", false, "add clipboardas another context")
    flag.Usage = func(){
        fmt.Printf("Usage: %s chat [-chat-file] [-prompt] [-clipboard-context] [query]\n", os.Args[0])
        flag.PrintDefaults()
    }
    flag.Parse(os.Args[2:])
    copilot, err := NewCopilotSession(NewInputWithInputFromArgs(flag.Args()))
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        return 1
    }

    if *prompt != ""{
        err = copilot.SetPrompt(*prompt)
    }

    if *chatFile != ""{
        err = copilot.SetChatFile(*chatFile)
    }


    if *clipBoardContext {
        text, err := clipboard.ReadAll()

        if err != nil {
            fmt.Fprintln(os.Stderr, err)
            return 1
        }

        copilot.chatHistory = append(copilot.chatHistory, Message{
            Role: "user",
            Content:text,
        })
    }

    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        return 1
    }

    err = copilot.Chat()
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        return 1
    }
    return 0
}

func Main() int {

    usage := `Usage: chatsh [COMMAND]
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
    chat    Chat with Copilot`


    if len(os.Args) < 2 {
        fmt.Println(usage)
        return 1
    }
    command := subcommands[os.Args[1]]

    if command == nil {
        fmt.Println("subcommand doesnt exist")
        return 1
    }

    switch command.Name() {
        case "setup":
            return MainAuthenticate()
        case "chat":
            return MainChat(command)
    }

    fmt.Fprintf(os.Stderr, usage)
    return 1

}
