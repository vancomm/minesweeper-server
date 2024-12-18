package main

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var (
	log = logrus.New()

	configPath string
	config     *Config

	pg *postgres

	jwtPrivateKey *rsa.PrivateKey
	jwtPublicKey  *rsa.PublicKey
)

func init() {
	const (
		defaultConfigPath = "/run/config.json"
		usage             = "config file path"
	)
	flag.StringVar(&configPath, "config", defaultConfigPath, usage)
	flag.StringVar(&configPath, "c", defaultConfigPath, usage+" (shorthand)")
}

func setupLogging() {
	logLevel := logrus.InfoLevel
	if config.Development() {
		logLevel = logrus.DebugLevel
	}
	log.SetLevel(logLevel)

	log.SetFormatter(&logrus.TextFormatter{ForceColors: true})
}

func setupPostgres(ctx context.Context) {
	var err error

	pg, err = NewPostgres(ctx, config.Postgres.DbUrl())
	if err != nil {
		log.Fatal("unable to create connection pool: ", err)
	}
	if err := pg.Ping(ctx); err != nil {
		log.Fatal("unable to ping database: ", err)
	}
}

func setupJwtKeys() {
	var err error

	privateKeyBytes, err := os.ReadFile(config.Jwt.PrivateKeyPath)
	if err != nil {
		log.Fatal("unable to read JWT private key: ", err)
	}
	jwtPrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
	if err != nil {
		log.Fatal("unable to parse JWT private key: ", err)
	}

	publicKeyBytes, err := os.ReadFile(config.Jwt.PublicKeyPath)
	if err != nil {
		log.Fatal("unable to read JWT public key: ", err)
	}
	jwtPublicKey, err = jwt.ParseRSAPublicKeyFromPEM(publicKeyBytes)
	if err != nil {
		log.Fatal("unable to parse JWT public key: ", err)
	}
}

func main() {
	mainCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt, syscall.SIGTERM,
	)
	defer stop()

	flag.Parse()

	if configBytes, err := os.ReadFile(configPath); err != nil {
		log.Fatalf("unable to read config %s: %s", configPath, err.Error())
	} else if err := json.Unmarshal(configBytes, config); err != nil {
		log.Fatalf("unable to parse config %s: %s", configPath, err.Error())
	}

	setupLogging()

	log.Info("starting up, mode = ", config.Mode)
	log.WithFields(config.Fields()).Debug("config")

	setupJwtKeys()

	setupPostgres(mainCtx)
	defer pg.Close()

	server := &http.Server{
		Addr:    config.Addr,
		Handler: buildHandler(),
		BaseContext: func(l net.Listener) context.Context {
			return mainCtx
		},
	}

	log.Infof("ready to serve @ %s", config.Addr)

	g, gCtx := errgroup.WithContext(mainCtx)
	g.Go(func() error {
		return server.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()
		return server.Shutdown(context.Background())
	})

	if err := g.Wait(); err != nil {
		log.Printf("exit reason: %s\n", err)
	}
}
