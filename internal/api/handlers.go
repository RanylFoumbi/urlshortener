package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	"urlshortener/cmd"
	"urlshortener/internal/models"
	"urlshortener/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm" // Pour gérer gorm.ErrRecordNotFound
)

// TODO Créer une variable ClickEventsChannel qui est un chan de type ClickEvent
// ClickEventsChannel est le channel global (ou injecté) utilisé pour envoyer les événements de clic
// aux workers asynchrones. Il est bufferisé pour ne pas bloquer les requêtes de redirection.

var ClickEventsChannel chan models.ClickEvent

// SetupRoutes configure toutes les routes de l'API Gin et injecte les dépendances nécessaires
func SetupRoutes(router *gin.Engine, linkService *services.LinkService) {
	// Le channel est initialisé ici.
	if ClickEventsChannel == nil {
		ClickEventsChannel = make(chan models.ClickEvent, viper.GetInt("analytics.buffer_size"))
	}
	// TODO : Route de Health Check , /health
	router.GET("/health", HealthCheckHandler)

	apiV1 := router.Group("/api/v1")
	{
		// POST /links
		apiV1.POST("/links", CreateShortLinkHandler(linkService))
		// GET /links/:shortCode/stats
		apiV1.GET("/links/:shortCode/stats", GetLinkStatsHandler(linkService))
	}
	// Route de Redirection (au niveau racine pour les short codes)
	router.GET("/:shortCode", RedirectHandler(linkService))
}

// HealthCheckHandler gère la route /health pour vérifier l'état du service.
func HealthCheckHandler(c *gin.Context) {
	// TODO  Retourner simplement du JSON avec un StatusOK, {"status": "ok"}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreateLinkRequest représente le corps de la requête JSON pour la création d'un lien.
type CreateLinkRequest struct {
	LongURL string `json:"long_url" binding:"required,url"` // 'binding:required' pour validation, 'url' pour format URL
}

// CreateShortLinkHandler gère la création d'une URL courte.
func CreateShortLinkHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateLinkRequest
		
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// TODO: Appeler le LinkService (CreateLink pour créer le nouveau lien.
		link, err := linkService.CreateLink(req.LongURL)
		if err != nil {
			// Si une erreur se produit, retourner un code HTTP 500 (Internal Server Error).
			log.Printf("Erreur lors de la création du lien: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Retourne le code court et l'URL longue dans la réponse JSON.
		c.JSON(http.StatusCreated, gin.H{
			"short_code":     link.ShortCode,
			"long_url":       link.LongURL,
			"full_short_url": cmd.Cfg.Server.BaseURL + link.ShortCode, // Utilise la base URL du serveur configurée
		})
	}
}

// RedirectHandler gère la redirection d'une URL courte vers l'URL longue et l'enregistrement asynchrone des clics.
func RedirectHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupère le shortCode de l'URL avec c.Param
		shortCode := c.Param("shortCode")

		link,err := linkService.GetLinkByShortCode(shortCode)

		if err != nil {
			// Si le lien n'est pas trouvé, retourner HTTP 404 Not Found.
			// Utiliser errors.Is et l'erreur Gorm
			if errors.Is(err, gorm.ErrRecordNotFound) { 
				// Utilisez errors.Is(err, gorm.ErrRecordNotFound) en production si l'erreur est wrappée
				c.JSON(http.StatusNotFound, gin.H{"error": "Lien introuvable"})
				return
			} else if errors.Is(err, gorm.ErrInvalidValue) {
				// Si l'erreur est une valeur invalide, retourner HTTP 400 Bad Request.
				c.JSON(http.StatusBadRequest, gin.H{"error": "Lien invalide"})
				return
			}
				// Gérer d'autres erreurs potentielles de la base de données ou du service
				log.Printf("Error retrieving link for %s: %v", shortCode, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		clickEvent := models.ClickEvent{
			LinkID:    link.ID,
			Timestamp: time.Now(),
			UserAgent: c.Request.UserAgent(),
			IPAddress: c.ClientIP(),
		}

		select {
		case ClickEventsChannel <- clickEvent:
			// Si l'envoi est réussi, on continue
		default:
			log.Printf("Warning: ClickEventsChannel is full, dropping click event for %s.", shortCode)
		}

		if link == nil || link.LongURL == "" {
			// Si le lien est introuvable ou l'URL longue est vide, retourner HTTP 404 Not Found.
			c.JSON(http.StatusNotFound, gin.H{"error": "Lien introuvable"})
			return
		}

		c.Redirect(http.StatusFound, link.LongURL)

	}
}

// GetLinkStatsHandler gère la récupération des statistiques pour un lien spécifique.
func GetLinkStatsHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {

		shortCode := c.Param("shortCode")

		link, err := linkService.GetLinkByShortCode(shortCode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Si le lien n'est pas trouvé, retourner HTTP 404 Not Found.
				c.JSON(http.StatusNotFound, gin.H{"error": "Lien introuvable"})
				return
			} else if errors.Is(err, gorm.ErrInvalidValue) {
				// Si l'erreur est une valeur invalide, retourner HTTP 400 Bad Request.
				c.JSON(http.StatusBadRequest, gin.H{"error": "Lien invalide"})
				return
			}
			// Gérer d'autres erreurs potentielles de la base de données ou du service
			log.Printf("Error retrieving link for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return

		}

		link, totalClicks, err := linkService.GetLinkStats(shortCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			log.Printf("Error retrieving link stats for %s: %v", shortCode, err)
			return
		}

		// Retourne les statistiques dans la réponse JSON.
		c.JSON(http.StatusOK, gin.H{
			"short_code":   link.ShortCode,
			"long_url":     link.LongURL,
			"total_clicks": totalClicks,
		})
	}
}
