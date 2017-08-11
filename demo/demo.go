// Provides a sample demo program using the filewatch package.
package main

import (
	"flag"
	"fmt"
	fw "github.com/0xABAD/filewatch"
	"time"
)

var (
	recurse  = flag.Bool("recurse", false, "watch files in directories recursively")
	interval = flag.Duration("interval", 2*time.Second, "how often to check for file modifications")
)

func main() {
	flag.Parse()
	file := flag.Arg(0)

	if file == "" {
		fmt.Println("No file or directory given to watch.\n")
		flag.PrintDefaults()
		return
	}

	done := make(chan struct{})
	updates, err := fw.Watch(done, file, *recurse, interval)
	if err != nil {
		fmt.Println(err)
		return
	}

	for update := range updates {
		fmt.Println("----------------")
		fmt.Println("Updates Received")
		fmt.Println("----------------")

		for _, u := range update {
			var change string

			switch {
			case u.WasRemoved:
				change = "Removed"
			case u.WasAdded:
				change = "Added"
			default:
				change = "Changed"
			}

			fmt.Printf("%s -- %s\n", u.AbsPath, change)
			fmt.Printf("\tIsDir:\t\t%v\n", u.Prev.IsDir())
			fmt.Printf("\tPrev, Size:\t%v, %v\n", u.Prev.ModTime(), u.Prev.Size())

			if u.Error != nil {
				fmt.Printf("\tError:\t\t%s\n", u.Error)
			} else if u.Next != nil {
				fmt.Printf("\tNext, Size:\t%v, %v\n", u.Next.ModTime(), u.Next.Size())
			}
		}
	}
	done <- struct{}{}
}
