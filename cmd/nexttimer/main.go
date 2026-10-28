package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/desirevolution/next-timer/internal/client"
	"github.com/desirevolution/next-timer/internal/config"
	"os"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "nexttimer:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		return client.Run([]string{"help"})
	}
	c, _, err := config.Read()
	if err != nil {
		return err
	}
	clientURL, err := config.ClientURL(c.URL)
	if err != nil {
		return err
	}
	flags := flag.NewFlagSet("nexttimer", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	url := flags.String("url", clientURL, "NextTimer-Server URL")
	detach := flags.Bool("detach", false, "Server-Timer starten und CLI sofort beenden")
	keepRunning := flags.Bool("keep-running", false, "CLI-gestarteten Timer beim Verlassen des Clients weiterlaufen lassen")
	local := flags.Bool("local", false, "Timer nur in diesem CLI/TUI-Prozess (ohne Server)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	client.SetServerURL(*url)
	commands := flags.Args()
	if len(commands) > 0 && commands[0] == "config" {
		switch {
		case len(commands) == 2 && commands[1] == "path":
			path, err := config.ClientConfigPath()
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		case len(commands) == 2 && commands[1] == "url":
			fmt.Println(*url)
			return nil
		case len(commands) == 2 && commands[1] == "check":
			fmt.Println("Konfiguration gültig")
			return nil
		case len(commands) == 2 && commands[1] == "show":
			b, err := json.MarshalIndent(c, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		case len(commands) == 2 && commands[1] == "init":
			path, err := config.Init()
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		case len(commands) == 3 && (commands[1] == "set-url" || commands[1] == "set"):
			path, err := config.SetClientURL(commands[2])
			if err != nil {
				return err
			}
			fmt.Printf("Server-URL gespeichert: %s (%s)\n", strings.TrimRight(commands[2], "/"), path)
			return nil
		default:
			return fmt.Errorf("Syntax: nexttimer config path|url|show|check|init|set-url <http://HOST:PORT>")
		}
	}
	return client.RunWithClientOptions(commands, client.Options{Detach: *detach, Local: *local, KeepRunning: *keepRunning})
}
