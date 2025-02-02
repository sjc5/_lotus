package glue

import "flag"

func (fw *Instance[AHD, SE, CEE]) Tasks() {
	devFlag := flag.Bool("dev", false, "Run Dev function")
	buildFlag := flag.Bool("build", false, "Run Build function")
	genFlag := flag.Bool("gen", false, "Run Gen function")

	flag.Parse()

	// Count how many flags are set to true
	flagCount := 0
	if *devFlag {
		flagCount++
	}
	if *buildFlag {
		flagCount++
	}
	if *genFlag {
		flagCount++
	}

	// Panic if no flags or multiple flags are set
	if flagCount == 0 {
		panic("No command flag specified. Use one of: -main, -dev, -build, -gen")
	}
	if flagCount > 1 {
		panic("Only one command flag can be specified at a time")
	}

	// Run the appropriate function based on the flag
	switch {
	case *devFlag:
		fw.Dev()
	case *buildFlag:
		fw.Build()
	case *genFlag:
		fw.Gen()
	}
}
