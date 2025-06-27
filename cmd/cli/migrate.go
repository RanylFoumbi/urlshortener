package cli

import (
	"fmt"
	"log"

	"urlshortener/cmd"
	"urlshortener/internal/models"

	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MigrateCmd représente la commande 'migrate'
var MigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Exécute les migrations de la base de données pour créer ou mettre à jour les tables.",
	Long: `Cette commande se connecte à la base de données configurée (SQLite)
et exécute les migrations automatiques de GORM pour créer les tables 'links' et 'clicks'
basées sur les modèles Go.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO : Charger la configuration chargée globalement via cmd.cfg
		cfg := cmd.Cfg

		// TODO 2: Initialiser la connexion à la base de données SQLite avec GORM.
		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("FATAL: Échec de la connexion à la base de données: %v", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Échec de l'obtention de la base de données SQL sous-jacente: %v", err)
		}

		// TODO Assurez-vous que la connexion est fermée après la migration.
		defer sqlDB.Close()

		// TODO 3: Exécuter les migrations automatiques de GORM.
		// Utilisez db.AutoMigrate() et passez-lui les pointeurs vers tous vos modèles.
		err = db.AutoMigrate(&models.Link{}, &models.Click{})
		if err != nil {
			log.Fatalf("FATAL: Erreur lors de l'exécution des migrations: %v", err)
		}

		// Pas touche au log
		fmt.Println("Migrations de la base de données exécutées avec succès.")
	},
}

func init() {
	// TODO : Ajouter la commande à RootCmd
	cmd.RootCmd.AddCommand(MigrateCmd)
}
