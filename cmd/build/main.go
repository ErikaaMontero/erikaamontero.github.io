// Command build genera el sitio estático de la campaña a partir de content/*.md,
// templates/*.html, static/ y fotos/. Salida: public/.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ErikaaMontero/erikaamontero.github.io/internal/site"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if err := site.Build(root, "public"); err != nil {
		fmt.Fprintf(os.Stderr, "build: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ sitio generado en public/")
}
