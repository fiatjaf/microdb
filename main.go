package main

import (
	"net/http"
	"os"
	"time"

	lightning "github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

var err error
var s Settings
var pg *sqlx.DB
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var spark *lightning.Client
var router = mux.NewRouter()

type Settings struct {
	Host        string `envconfig:"HOST" default:"0.0.0.0"`
	Port        string `envconfig:"PORT" required:"true"`
	ServiceURL  string `envconfig:"SERVICE_URL" required:"true"`
	PostgresURL string `envconfig:"DATABASE_URL" required:"true"`
	SparkURL    string `envconfig:"SPARK_URL" required:"true"`
	SparkToken  string `envconfig:"SPARK_TOKEN" required:"true"`
}

func main() {
	err = envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	// postgres connection
	pg, err = sqlx.Connect("postgres", s.PostgresURL)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't connect to postgres")
	}

	// spark caller and listener
	spark = &lightning.Client{
		SparkURL:    s.SparkURL,
		SparkToken:  s.SparkToken,
		CallTimeout: time.Second * 3,
	}
	go sparkoListener()

	// routes
	router.Path("/fund/{bucket}").Methods("GET").HandlerFunc(fundBucket)
	router.Path("/_/{bucket}").Methods("GET").HandlerFunc(getBucketStatus)
	router.Path("/{bucket}").Methods("GET", "POST", "PUT", "PATCH", "DELETE").
		HandlerFunc(handleBucketAccess)
	router.PathPrefix("/{bucket}/").Methods("GET", "POST", "PUT", "PATCH", "DELETE").
		HandlerFunc(handleBucketAccess)
	router.PathPrefix("/").Handler(br.Serve("public"))

	// start http server
	log.Info().Str("host", s.Host).Str("port", s.Port).Msg("listening")
	srv := &http.Server{
		Handler:      router,
		Addr:         s.Host + ":" + s.Port,
		WriteTimeout: 300 * time.Second,
		ReadTimeout:  300 * time.Second,
	}
	err = srv.ListenAndServe()
	if err != nil {
		log.Error().Err(err).Msg("error serving http")
	}
}
