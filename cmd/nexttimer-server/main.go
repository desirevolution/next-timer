package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/desirevolution/next-timer/internal/config"
	"github.com/desirevolution/next-timer/internal/daemon"
	"github.com/desirevolution/next-timer/internal/server"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "nexttimer-server:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Println("nexttimer-server [serve] [--listen ...] | service install|uninstall|start|stop|restart|status | config path|show|check|init")
		return nil
	}
	if len(args) > 0 && args[0] == "config" {
		return configCommand(args[1:])
	}
	if len(args) > 0 && args[0] == "service" {
		if len(args) != 2 {
			return errors.New("Syntax: nexttimer service install|uninstall|start|stop|restart|status")
		}
		return daemon.Service(args[1])
	}
	c, _, err := config.Read()
	if err != nil {
		return err
	}
	if len(args) > 0 && args[0] == "serve" {
		args = args[1:]
	}
	return serve(c, args)
}
func configCommand(args []string) error {
	if len(args) != 1 {
		return errors.New("Syntax: nexttimer config path|show|check|init")
	}
	switch args[0] {
	case "init":
		path, err := config.Init()
		if err != nil {
			return err
		}
		fmt.Println("Konfiguration erstellt:", path)
		return nil
	case "path":
		path, err := config.ConfigPath()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	case "check", "show":
		c, path, err := config.Read()
		if err != nil {
			return err
		}
		if args[0] == "check" {
			fmt.Println("OK:", path)
			return nil
		}
		b, err := json.MarshalIndent(c, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	return errors.New("Syntax: nexttimer config path|show|check|init")
}
func serve(c config.Config, args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	listen := flags.String("listen", c.Listen, "HTTP-Adresse")
	data := flags.String("data", c.Data, "Vorlagen-Datei")
	awtrix := flags.String("awtrix", c.AWTRIX, "AWTRIX-NG-HTTP-URL")
	alarmWebhook := flags.String("alarm-webhook", c.AlarmWebhook, "Optionaler HTTP-Webhook bei timer.finished")
	finishEffect := flags.String("awtrix-finish-effect", c.AWTRIXFinishEffect, "AWTRIX-Effekt bei Timer-Ende (z.B. Fireworks, ColorWaves, none)")
	finishEffectSpeed := flags.Float64("awtrix-finish-effect-speed", c.AWTRIXFinishEffectSpeed, "AWTRIX Effektgeschwindigkeit (0.1..10)")
	finishDuration := flags.Duration("awtrix-finish-duration", mustDuration(c.AWTRIXFinishDuration), "AWTRIX Fertig-Meldung anzeigen (z.B. 10s)")
	awtrixSyncLead := flags.Duration("awtrix-sync-lead", mustDuration(c.AWTRIXSyncLead), "AWTRIX Anzeigeversatz ausgleichen (0..500ms)")
	ttl := flags.Duration("finished-ttl", mustDuration(c.FinishedTTL), "Aufbewahrung fertiger Timer")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *ttl < 5*time.Second {
		return errors.New("finished-ttl mindestens 5s")
	}
	if math.IsNaN(*finishEffectSpeed) || math.IsInf(*finishEffectSpeed, 0) || *finishEffectSpeed < 0.1 || *finishEffectSpeed > 10 {
		return errors.New("awtrix-finish-effect-speed muss zwischen 0.1 und 10 liegen")
	}
	if *finishDuration < time.Second || *finishDuration > 5*time.Minute {
		return errors.New("awtrix-finish-duration muss zwischen 1s und 5m liegen")
	}
	if *awtrixSyncLead < 0 || *awtrixSyncLead > 500*time.Millisecond {
		return errors.New("awtrix-sync-lead muss zwischen 0ms und 500ms liegen")
	}
	service, err := server.NewService(*data, *ttl)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				service.Tick()
			}
		}
	}()
	if strings.TrimSpace(*alarmWebhook) != "" {
		go server.NewAlarmWebhook(*alarmWebhook).Run(ctx, service)
		log.Print("Alarm-Webhook aktiviert (nur timer.finished)")
	}
	if strings.TrimSpace(*awtrix) != "" {
		a := server.NewAwtrix(*awtrix)
		a.SetFinishEffect(*finishEffect)
		a.SetFinishOptions(*finishEffectSpeed, *finishDuration)
		a.SetDisplayLead(*awtrixSyncLead)
		go a.Run(ctx, service)
		log.Printf("AWTRIX: %s", *awtrix)
	}
	httpServer := &http.Server{Addr: *listen, Handler: (server.API{Service: service}).Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(c)
	}()
	log.Printf("NextTimer http://%s", *listen)
	err = httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
func mustDuration(s string) time.Duration { d, _ := time.ParseDuration(s); return d }
