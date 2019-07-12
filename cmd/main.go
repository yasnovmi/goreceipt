package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/yasnov/goreceipt"
	"github.com/yasnov/goreceipt/api/dal"
	"github.com/yasnov/goreceipt/api/dataloaders"
	"github.com/yasnov/goreceipt/api/resolver"
	"github.com/yasnov/goreceipt/config"
	"github.com/yasnov/goreceipt/loader"

	"github.com/99designs/gqlgen/handler"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

const defaultPort = "8080"

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Fatal("No .env file found")
	}
	config.Config = config.New()
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	db, err := dal.Connect()
	if err != nil {
		panic(err)
	}

	res := resolver.NewResolver(db)
	go loader.StartLoader(res)

	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	//router.Use(middleware.Logger)
	//router.Use(logger.NewStructuredLogger())

	queryHandler := corsAccess(handler.GraphQL(goreceipt.NewExecutableSchema(resolver.NewRootResolvers(res)),
		handler.WebsocketKeepAliveDuration(0),
		handler.WebsocketUpgrader(websocket.Upgrader{
			CheckOrigin: func(request *http.Request) bool {
				return true
			},
			HandshakeTimeout: 10 * time.Second,
		}),
	))
	router.Handle("/", handler.Playground("GraphQL playground", "/query"))
	router.Handle("/query", dataloaders.LoaderMiddleware(db, queryHandler))
	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func corsAccess(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Access-Control-Allow-Origin", "*")
		response.Header().Set("Access-Control-Allow-Credentials", "true")
		response.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		response.Header().Set("Access-Control-Allow-Headers", "Accept, X-Requested-With, Content-Type, Authorization")
		next(response, request)
	})
}
