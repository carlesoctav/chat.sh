package chatsh

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type AuthResponse struct {
    DeviceCode string `json:"device_code"` 
    UserCode string `json:"user_code"` 
    VerificationUri string `json:"verification_uri"`
    ExpiresIn int `json:"expires_in"`
    Interval int `json:"interval"`
}

type Cache struct {
    GithubToken string `json:"github_token"`
    CopilotToken string `json:"copilot_token"`
    ExpiresAt string `json:"expires_at"`
}


func CacheTokenNewAuth(token string) error {
    homeDir, err := os.UserHomeDir()

    if err != nil {
        return err
    }

    filePath := filepath.Join(homeDir, ".config", ".chatsh")
    f, err := os.Create(filePath)

    if err != nil {
        return err
    }
    cache := Cache{
        GithubToken: token,
        ExpiresAt: "",
        CopilotToken: "",
    }


    defer f.Close()

    return json.NewEncoder(f).Encode(cache)
}

func VerifyGithub(deviceCode string) error {
    url := "https://github.com/login/oauth/access_token"
    client := http.DefaultClient
    body := map[string]string{
        "client_id": "Iv1.b507a08c87ecfe98",
        "device_code":deviceCode,
        "grant_type": "urn:ietf:params:oauth:grant-type:device_code",
    }

    loginHeader := map[string]string{
        "accept": "application/json",
        "content-type": "application/json",
        "editor-version": "Neovim/0.9.2",
        "editor-plugin-version": "chat.sh/0.1",
        "user-agent": "GithubCopilot/1.133.0",
    }

    jsonBody, err := json.Marshal(body)

    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
    
    for k, v := range loginHeader {
        req.Header.Set(k, v)
    }

    if err != nil {
        return err
    }

    resp, err := client.Do(req)

    if err != nil {
        return err
    }

    var results map[string]string

    data, err := io.ReadAll(resp.Body)

    if err != nil {
        return err
    }

    err = json.Unmarshal(data, &results)
    if _, ok := results["access_token"]; !ok{
        return errors.New("please do the instruction")
    }

    err = CacheTokenNewAuth(results["access_token"])

    if err != nil {
        return err
    }

    return nil
}

func githubAuthenticate() error {
    req, err := githubRequestAuth()

    if err != nil {
        return err
    }

    fmt.Printf("please visit %v\nand enter %v code\n", req.VerificationUri, req.UserCode)

    for {
        err := VerifyGithub(req.DeviceCode)
        if err.Error() == "please do the instruction"{
            println("please do the instruction\n")
            time.Sleep(5 * time.Second)
            continue
        }

            if err == nil {
                break
            }
    }

    return nil
    
}

func githubRequestAuth() (AuthResponse, error){
    url := "https://github.com/login/device/code"
    client := &http.Client{}

    loginHeader := map[string]string{
        "accept": "application/json",
        "content-type": "application/json",
        "editor-version": "Neovim/0.9.2",
        "editor-plugin-version": "chat.sh/0.1",
        "user-agent": "GithubCopilot/1.133.0",
    }

    body := map[string]string{
        "client_id": "Iv1.b507a08c87ecfe98",
        "scope": "read:user",
    }

    jsonBody, err := json.Marshal(body)

    if err != nil {
        return AuthResponse{}, err
    }

    req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))

    if err != nil {
        return AuthResponse{}, err
    }
    for k, v:= range loginHeader{
        req.Header.Set(k,v)
    }

    resp, err := client.Do(req)

    if err != nil {
        return AuthResponse{}, err
    }

    var results AuthResponse
    data, err := io.ReadAll(resp.Body)
    err = json.Unmarshal(data, &results)

    if err != nil {
        return AuthResponse{}, err
    }

    return results, nil
}
