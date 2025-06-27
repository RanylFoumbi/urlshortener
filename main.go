package main

import (
	"urlshortener/cmd"
	_ "urlshortener/cmd/cli"    // Importe le package 'cli' pour que ses init() soient exécutés
	_ "urlshortener/cmd/server" // Importe le package 'server' pour que ses init() soient exécutés
)

func main() {
	// TODO Exécute la commande racine de Cobra.
	cmd.Execute()
}
