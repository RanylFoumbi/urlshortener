package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"urlshortener/cmd"
	"urlshortener/internal/api"
	"urlshortener/internal/monitor"
	"urlshortener/internal/repository"
	"urlshortener/internal/services"
	"urlshortener/internal/workers"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// RunServerCmd représente la commande 'run-server' de Cobra.
// C'est le point d'entrée pour lancer le serveur de l'application.
var RunServerCmd = &cobra.Command{
	Use:   "run-server",
	Short: "Lance le serveur API de raccourcissement d'URLs et les processus de fond.",
	Long: `Cette commande initialise la base de données, configure les APIs,
démarre les workers asynchrones pour les clics et le moniteur d'URLs,
puis lance le serveur HTTP.`,
	Run: func(cmdCobra *cobra.Command, args []string) {
		// TODO : Charger la configuration chargée globalement via cmd.cfg
		cfg := cmd.Cfg
		if cfg == nil {
			log.Fatalf("Erreur : configuration non chargée.")
		}

		// TODO : Initialiser la connexion à la base de données SQLite avec GORM.
		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Erreur de connexion DB : %v", err)
		}

		// TODO : Initialiser les repositories.
		linkRepo := repository.NewLinkRepository(db)
		clickRepo := repository.NewClickRepository(db)

		log.Println("Repositories initialisés.")

		// TODO : Initialiser les services métiers.
		linkService := services.NewLinkService(linkRepo)
		clickService := services.NewClickService(clickRepo)

		// Laissez le log
		log.Println("Services métiers initialisés.")

		// TODO : Initialiser le channel ClickEventsChannel (api/handlers) des événements de clic et lancer les workers (StartClickWorkers).
		api.ClickEventsChannel = make(chan services.ClickEvent, cfg.Analytics.BufferSize)
		go workers.StartClickWorker(api.ClickEventsChannel, clickService)

		// TODO : Remplacer les XXX par les bonnes variables
		log.Printf("Channel d'événements de clic initialisé avec un buffer de %d. %d worker(s) de clics démarré(s).",
			cfg.Analytics.BufferSize, 1)

		// TODO : Initialiser et lancer le moniteur d'URLs.
		monitorInterval := time.Duration(cfg.Monitor.IntervalMinutes) * time.Minute
		urlMonitor := monitor.NewUrlMonitor(linkRepo, monitorInterval)
		go urlMonitor.Start()
		log.Printf("Moniteur d'URLs démarré avec un intervalle de %v.", monitorInterval)

		// TODO : Configurer le routeur Gin et les handlers API.
		router := gin.Default()
		api.SetupRoutes(router, linkService)
		log.Println("Routes API configurées.")

		// Créer le serveur HTTP Gin
		serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
		srv := &http.Server{
			Addr:    serverAddr,
			Handler: router,
		}

		// TODO : Démarrer le serveur Gin dans une goroutine anonyme pour ne pas bloquer.
		go func() {
			log.Printf("Serveur démarré sur %s\n", serverAddr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Erreur au lancement du serveur HTTP: %v", err)
			}
		}()

		// Gére l'arrêt propre du serveur (graceful shutdown).
		// Créez un channel pour les signaux OS (SIGINT, SIGTERM).
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Bloquer jusqu'à ce qu'un signal d'arrêt soit reçu.
		<-quit
		log.Println("Signal d'arrêt reçu. Arrêt du serveur...")

		// Arrêt propre du serveur HTTP avec un timeout.
		log.Println("Arrêt en cours... Donnez un peu de temps aux workers pour finir.")
		time.Sleep(5 * time.Second)

		log.Println("Serveur arrêté proprement.")
	},
}

func init() {
	// TODO : ajouter la commande
	cmd.RootCmd.AddCommand(RunServerCmd)
}
