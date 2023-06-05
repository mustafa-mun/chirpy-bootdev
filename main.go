package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/mustafa-mun/chirpy-bootdev/internal/controller"
	"github.com/mustafa-mun/chirpy-bootdev/internal/handler"
	"github.com/mustafa-mun/chirpy-bootdev/internal/sys"
)

func main() {

	sys.EnableDebugMode()
	sys.LoadDotenv()
	controller.InitDB()
	
	r := chi.NewRouter()
	apiRouter := chi.NewRouter()
	adminRouter := chi.NewRouter()
	corsMux := handler.MiddlewareCors(r)
	apiCfg := &controller.ApiConfig{FileserverHits: 0, JwtSecret: os.Getenv("JWT_SECRET")}

	fsHandler := apiCfg.MiddlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	r.Handle("/app", fsHandler)
	r.Handle("/app/*", fsHandler)
	r.Mount("/api", apiRouter)
	r.Mount("/admin", adminRouter)

	adminRouter.Get("/metrics", apiCfg.MetricsHandler)

	apiRouter.Get("/healthz", apiCfg.HealthzHandler)
	apiRouter.Get("/chirps", apiCfg.GetChirpsHandler)
	apiRouter.Get("/chirps/{chirpId}", apiCfg.GetSingleChirpHandler)

	apiRouter.Post("/chirps", apiCfg.PostChirpHandler)
	apiRouter.Post("/users", apiCfg.PostUserHandler)
	apiRouter.Post("/login", apiCfg.LoginHandler)
	apiRouter.Post("/polka/webhooks", apiCfg.PolkaWebhooksHandler) // polka payment handling 

	apiRouter.Post("/refresh", apiCfg.RefreshTokenHandler) // Refresh access token
	apiRouter.Post("/revoke", apiCfg.RevokeTokenHandler) // Revoke refresh token


	apiRouter.Put("/users", apiCfg.UpdateUserHandler)

	apiRouter.Delete("/chirps/{chirpID}", apiCfg.DeleteChirpHandler)

	server := &http.Server{
		Addr:    ":" + os.Getenv("PORT"),
		Handler: corsMux,
	}
	server.ListenAndServe()
}







