package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
)

var pokemonTemplate *template.Template
var pokestopTemplate *template.Template
var gymTemplate *template.Template

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe /healthz on the local server and exit 0 (healthy) or 1")
	flag.Parse()

	cfg, err := Load()
	if *healthcheck {
		if err != nil {
			os.Exit(1)
		}
		os.Exit(runHealthcheck(cfg.Port))
	}
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	config = cfg

	//	templateStr := "https://maps.google.com/maps?q={{.lat}},{{.lon}}"

	pokemonTemplate = template.New("pokemon")
	pokestopTemplate = template.New("pokestop")
	gymTemplate = template.New("gym")

	for _, t := range config.Pokemon {
		pokemonTemplate, err = pokemonTemplate.New(t.Name).Parse(t.Url)
		if err != nil {
			panic(err)
		}
	}

	for _, t := range config.Pokestop {
		pokestopTemplate, err = pokestopTemplate.New(t.Name).Parse(t.Url)
		if err != nil {
			panic(err)
		}
	}

	for _, t := range config.Gym {
		gymTemplate, err = gymTemplate.New(t.Name).Parse(t.Url)
		if err != nil {
			panic(err)
		}
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/pokemon/:pokemon_id/:template", GetPokemon)
	r.GET("/pokestop/:pokestop_id/:template", GetPokestop)
	r.GET("/gym/:gym_id/:template", GetGym)
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: r,
	}
	go func() {
		fmt.Printf("%s [] Starting server on port %d\n", time.Now().Format(config.TimestampFormat), config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	stop()

	fmt.Printf("%s [] Shutting down\n", time.Now().Format(config.TimestampFormat))
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
}

func runHealthcheck(port int) int {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/healthz", port))
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
