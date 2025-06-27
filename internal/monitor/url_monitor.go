package monitor

import (
	"log"
	"net/http"
	"sync" // Pour protéger l'accès concurrentiel à knownStates
	"time"

	_ "urlshortener/internal/models"   // Importe les modèles de liens
	"urlshortener/internal/repository" // Importe le repository de liens
)

// UrlMonitor gère la surveillance périodique des URLs longues.
type UrlMonitor struct {
	linkRepo    repository.LinkRepository // Pour récupérer les URLs à surveiller
	interval    time.Duration             // Intervalle entre chaque vérification (ex: 5 minutes)
	knownStates map[uint]bool             // État connu de chaque URL: map[LinkID]estAccessible (true/false)
	mu          sync.Mutex                // Mutex pour protéger l'accès concurrentiel à knownStates
}

// NewUrlMonitor crée et retourne une nouvelle instance de UrlMonitor.
func NewUrlMonitor(linkRepo repository.LinkRepository, interval time.Duration) *UrlMonitor {
	return &UrlMonitor{
		linkRepo:    linkRepo,
		interval:    interval,
		knownStates: make(map[uint]bool),
		mu:          sync.Mutex{},
	}
}

// Start lance la boucle de surveillance périodique des URLs.
// Cette fonction est conçue pour être lancée dans une goroutine séparée.
func (m *UrlMonitor) Start() {
	log.Printf("[MONITOR] Démarrage du moniteur d'URLs avec un intervalle de %v...", m.interval)
	ticker := time.NewTicker(m.interval) // Crée un ticker qui envoie un signal à chaque intervalle
	defer ticker.Stop()                  // S'assure que le ticker est arrêté quand Start se termine

	// Exécute une première vérification immédiatement au démarrage
	m.checkUrls()

	// Boucle principale du moniteur, déclenchée par le ticker
	for range ticker.C {
		m.checkUrls()
	}
}

// checkUrls effectue une vérification de l'état de toutes les URLs longues enregistrées.
func (m *UrlMonitor) checkUrls() {
	log.Println("[MONITOR] Lancement de la vérification de l'état des URLs...")

	// Gérer l'erreur si la récupération échoue.
	links, err := m.linkRepo.GetAllLinks()
	if err != nil {
		log.Printf("[MONITOR] ERREUR lors de la récupération des liens pour la surveillance : %v", err)
	}

	for _, link := range links {
		// Vérifier l'accessibilité de l'URL
		currentState := m.isUrlAccessible(link.LongURL)

		// Protéger l'accès à la map 'knownStates' car 'checkUrls' peut être exécuté concurremment
		m.mu.Lock()
		previousState, exists := m.knownStates[link.ID]
		m.knownStates[link.ID] = currentState
		m.mu.Unlock()

		// Si c'est la première vérification pour ce lien, on initialise l'état sans notifier.
		if !exists {
			log.Printf("[MONITOR] État initial pour le lien %s (%s) : %s",
				link.ShortCode, link.LongURL, formatState(currentState))
			continue
		}

		// Notifier si l'état a changé
		if previousState != currentState {
			log.Printf("[NOTIFICATION] Le lien %s (%s) est passé de %s à %s !",
				link.ShortCode,
				link.LongURL,
				formatState(previousState),
				formatState(currentState))
		}
	}
	log.Println("[MONITOR] Vérification de l'état des URLs terminée.")
}

// isUrlAccessible effectue une requête HTTP HEAD pour vérifier l'accessibilité d'une URL.
func (m *UrlMonitor) isUrlAccessible(url string) bool {
	// Création d'un client HTTP avec timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Création de la requête HEAD
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Printf("[MONITOR] Erreur de création de la requête pour '%s': %v", url, err)
		return false
	}

	// Exécution de la requête
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[MONITOR] Erreur d'accès à l'URL '%s': %v", url, err)
		return false
	}
	defer resp.Body.Close() // Fermeture du corps de la réponse

	// Déterminer l'accessibilité basée sur le code de statut HTTP
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// formatState est une fonction utilitaire pour rendre l'état plus lisible dans les logs.
func formatState(accessible bool) string {
	if accessible {
		return "ACCESSIBLE"
	}
	return "INACCESSIBLE"
}
