package repository

import (
	"urlshortener/internal/models"

	"gorm.io/gorm"
)

// LinkRepository est une interface qui définit les méthodes d'accès aux données
// pour les opérations CRUD sur les liens.
type LinkRepository interface {
	CreateLink(link *models.Link) error
	GetLinkByShortCode(shortCode string) (*models.Link, error)
	GetAllLinks() ([]models.Link, error)
	CountClicksByLinkID(linkID uint) (int, error)
}

type GormLinkRepository struct {
	db *gorm.DB
}

// NewLinkRepository crée et retourne une nouvelle instance de GormLinkRepository.
// Cette fonction retourne *GormLinkRepository, qui implémente l'interface LinkRepository.
func NewLinkRepository(db *gorm.DB) *GormLinkRepository {
	return &GormLinkRepository{
		db: db,
	}
}

func (r *GormLinkRepository) CreateLink(link *models.Link) error {
	return r.db.Create(link).Error
}

// GetLinkByShortCode récupère un lien de la base de données en utilisant son shortCode.
// Il renvoie gorm.ErrRecordNotFound si aucun lien n'est trouvé avec ce shortCode.
func (r *GormLinkRepository) GetLinkByShortCode(shortCode string) (*models.Link, error) {
	var link models.Link
	err := r.db.Where("short_code = ?", shortCode).First(&link).Error

	if err != nil {
		return nil, err
	}
	return &link, nil
}

// GetAllLinks récupère tous les liens de la base de données.
// Cette méthode est utilisée par le moniteur d'URLs.
func (r *GormLinkRepository) GetAllLinks() ([]models.Link, error) {
	var links []models.Link
	err := r.db.Find(&links).Error

	if err != nil {
		return nil, err
	}
	return links, nil
}

// CountClicksByLinkID compte le nombre total de clics pour un ID de lien donné.
func (r *GormLinkRepository) CountClicksByLinkID(linkID uint) (int, error) {
	var count int64
	err := r.db.Model(&models.Click{}).Where("link_id = ?", linkID).Count(&count).Error

	if err != nil {
		return 0, err
	}
	return int(count), nil
}
