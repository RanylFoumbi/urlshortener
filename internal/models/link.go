package models

import "time"

// TODO : Créer la struct Link
// Link représente un lien raccourci dans la base de données.
// Les tags `gorm:"..."` définissent comment GORM doit mapper cette structure à une table SQL.
// ID qui est une primaryKey
// Shortcode : doit être unique, indexé pour des recherches rapide (voir doc), taille max 10 caractères
// LongURL : doit pas être null
// CreateAt : Horodatage de la créatino du lien

type Link struct {
	ID        uint      `gorm:"primaryKey"`
	Shortcode string    `gorm:"uniqueIndex;size:10"`
	LongURL   string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	clicks    []Click
}
