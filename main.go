package main

import (
	"context"
	"crypto/rsa"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"github.com/snowzach/rotatefilehook"
	"golang.org/x/sync/errgroup"
)

var log = logrus.New()

var (
	configPath string
	config     Config
)

func init() {
	const (
		defaultConfigPath = "./config.json"
		usage             = "config file path"
	)
	flag.StringVar(&configPath, "config", defaultConfigPath, usage)
	flag.StringVar(&configPath, "c", defaultConfigPath, usage+" (shorthand)")
}

var logPath string

func init() {
	const (
		defaultLogPath = "./log.jsonl"
		usage          = "log file path"
	)
	flag.StringVar(&logPath, "log-file", defaultLogPath, usage)
	flag.StringVar(&logPath, "l", defaultLogPath, usage+" (shorthand)")
}

func setupLogging() {
	logLevel := logrus.InfoLevel
	if config.Development() {
		logLevel = logrus.DebugLevel
	}
	log.SetLevel(logLevel)

	log.SetFormatter(&logrus.TextFormatter{ForceColors: true})

	if hook, err := rotatefilehook.NewRotateFileHook(
		rotatefilehook.RotateFileConfig{
			Filename:   logPath,
			MaxSize:    5,
			MaxBackups: 7,
			MaxAge:     28,
			Level:      logLevel,
			Formatter:  &logrus.JSONFormatter{},
		}); err != nil {
		log.Fatal("unable to set set up rotating file logger: ", err)
	} else {
		log.Hooks.Add(hook)
	}
}

var pg *postgres

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

// ssh-keygen -t rsa -m pem -f jwt-private-key.pem
// openssl rsa -in jwt-private-key.pem -pubout -out jwt-public-key.pem

var (
	jwtPrivateKey *rsa.PrivateKey
	jwtPublicKey  *rsa.PublicKey
)

func setupJwtKeys() {
	privateKeyBytes, err := os.ReadFile(config.Jwt.PrivateKeyPath)
	if err != nil {
		log.Fatal("unable to read JWT private key: ", err)
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
	if err != nil {
		log.Fatal("unable to parse JWT private key: ", err)
	}
	publicKeyBytes, err := os.ReadFile(config.Jwt.PublicKeyPath)
	if err != nil {
		log.Fatal("unable to read JWT public key: ", err)
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyBytes)
	if err != nil {
		log.Fatal("unable to parse JWT public key: ", err)
	}
	jwtPrivateKey = privateKey
	jwtPublicKey = publicKey
}

func main() {
	mainCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt, syscall.SIGTERM,
	)
	defer stop()

	flag.Parse()

	if err := ReadConfig(configPath, &config); err != nil {
		log.Fatalf("unable to read config from %s: %s", configPath, err.Error())
	}

	setupLogging()

	log.WithFields(config.Fields()).Debug("config")
	log.Info("starting up, mode = ", config.Mode)

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
