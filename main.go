package main

import (
	"log"

	"github.com/murakmii/gotermda/ui"
)

func main() {
	webUI := ui.NewWebUI()
	if err := webUI.ListenAndServe(":8080"); err != nil {
		log.Fatalf("web ui error: %s", err)
	}
}
