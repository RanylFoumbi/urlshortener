package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"gorm.io/gorm" // Nécessaire pour la gestion spécifique de gorm.ErrRecordNotFound
	"log"

	"urlshortener/internal/models"
	"urlshortener/internal/repository" // Importe le package repository
)

// Définition du jeu de caractères pour la génération des codes courts.
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// TODO Créer la struct
// LinkService est une structure qui g fournit des méthodes pour la logique métier des liens.
// Elle détient linkRepo qui est une référence vers une interface LinkRepository.
// IMPORTANT : Le champ doit être du type de l'interface (non-pointeur).

type LinkService struct {
	linkRepo repository.LinkRepository
}

// NewLinkService crée et retourne une nouvelle instance de LinkService.
func NewLinkService(linkRepo repository.LinkRepository) *LinkService {
	return &LinkService{
		linkRepo: linkRepo,
	}
}

// GenerateShortCode est une méthode rattachée à LinkService
// Elle génère un code court aléatoire d'une longueur spécifiée. Elle prend une longueur en paramètre et retourne une string et une erreur
// Il utilise le package 'crypto/rand' pour éviter la prévisibilité.
// Je vous laisse chercher un peu :) C'est faisable en une petite dizaine de ligne
func (s *LinkService) GenerateShortCode() (string, error) {
	length := 6
	bytes := make([]byte, length)

	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("erreur lors de la génération aléatoire: %w", err)
	}

	shortCode := make([]byte, length)
	for i := 0; i < length; i++ {
		shortCode[i] = charset[bytes[i]%byte(len(charset))]
	}

	return string(shortCode), nil
}

// CreateLink crée un nouveau lien raccourci.
// Il génère un code court unique, puis persiste le lien dans la base de données.
func (s *LinkService) CreateLink(longURL string) (*models.Link, error) {
	// TODO 1: Implémenter la logique de retry pour générer un code court unique.
	// Essayez de générer un code, vérifiez s'il existe déjà en base, et retentez si une collision est trouvée.
	// Limitez le nombre de tentatives pour éviter une boucle infinie.
	var shortCode string
	var err error

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {

		shortCode, err = s.GenerateShortCode()
		_, err = s.linkRepo.GetLinkByShortCode(shortCode)

		if err != nil {
			// Si l'erreur est 'record not found' de GORM, cela signifie que le code est unique.
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			// Si c'est une autre erreur de base de données, retourne l'erreur.
			return nil, fmt.Errorf("database error checking short code uniqueness: %w", err)
		} else {
			if i == maxRetries-1 {
				return nil, errors.New("maximum number of retries reached")
			}
		}

		// Si aucune erreur (le code a été trouvé), cela signifie une collision.
		log.Printf("Short code '%s' already exists, retrying generation (%d/%d)...", shortCode, i+1, maxRetries)
		// La boucle continuera pour générer un nouveau code.
	}

	link := models.Link{
		Shortcode: shortCode,
		LongURL:   longURL,
	}
	err = s.linkRepo.CreateLink(&link)
	if err != nil {
		log.Printf("Error creating link: %v", err)
		return nil, err
	}

	return &link, nil
}

// GetLinkByShortCode récupère un lien via son code court.
// Il délègue l'opération de recherche au repository.
func (s *LinkService) GetLinkByShortCode(shortCode string) (*models.Link, error) {
	return s.linkRepo.GetLinkByShortCode(shortCode)
}

// GetLinkStats récupère les statistiques pour un lien donné (nombre total de clics).
// Il interagit avec le LinkRepository pour obtenir le lien, puis avec le ClickRepository
func (s *LinkService) GetLinkStats(shortCode string) (*models.Link, int, error) {
	var err error
	var link *models.Link
	var count int

	link, err = s.linkRepo.GetLinkByShortCode(shortCode)
	if err != nil {
		return nil, 0, fmt.Errorf("error retrieving link: %w", err)
	}

	count, err = s.linkRepo.CountClicksByLinkID(link.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("error retrieving link stats: %w", err)
	}

	return link, count, nil
}
