package main

import (
	"flag"
	"fmt"
	"log"
	"bus-app/utils"
)

func main() {
	var (
		templatesDir = flag.String("dir", "templates", "Directory containing HTML templates")
		mode        = flag.String("mode", "all", "Update mode: 'all' for specific pages, 'simple' for all HTML files")
	)
	flag.Parse()

	fmt.Printf("Starting design update in %s mode for directory: %s\n", *mode, *templatesDir)

	var err error
	switch *mode {
	case "all":
		err = utils.UpdateAllPagesDesign(*templatesDir)
	case "simple":
		err = utils.UpdatePagesDesignSimple(*templatesDir)
	default:
		log.Fatalf("Invalid mode: %s. Use 'all' or 'simple'", *mode)
	}

	if err != nil {
		log.Fatalf("Error updating design: %v", err)
	}
}
