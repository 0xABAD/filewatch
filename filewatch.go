// Package filewatch provides a utility to watch a file or directory for changes.
package filewatch

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Update struct {
	AbsPath    string      // absolute file path of the file that was updated
	Prev       os.FileInfo // previous file info from the last update
	Next       os.FileInfo // file info received for this update
	Error      error       // an error if one occurred
	WasRemoved bool        // was removed from last update
	WasAdded   bool        // was added since last update
}

// Watches a file or the contents of a directory for changes.
//
// This function can watch an individual file or the contents of directory
// on a set interval and any changes will be aggregated to a slice and sent to the
// returned channel.  Path  will be resolved to its absolute path and failure to do
// so will return an  error.  If path is a directory then that directory along with
// all of its contents will be watched and when recurse is set to true then any files
// and directories within sub-directories of path will be watched.
//
// The default interval to check for updates is every two seconds unless interval
// is non nil which then that is used.  If interval is non nil and is less than
// or equal to zero than an error is returned.  Watch will continue to scan for
// file changes on the set interval until a value is received from the done channel
// or it is closed.  Changes are pushed to the returned channel only when actual changes
// are detected.  In other words, Watch will never send an empty slice unless the done
// channel is closed which causes the returned channel to be closed.
//
// If an error occurs when checking for a change of a file or directory during an
// update scan then that error is attached to the Update.Error field and Watch will
// continue scanning on its regular interval.  If the error was caused due to the file
// or directory being deleted then the error will still be attached to Update.Error
// but Update.WasRemoved will also be set to true and the file or directory will no
// longer be watched.
//
// Since Watch uses filepath.Walk to setup which files to watch means that it
// will not follow symbolic links even when recurse is true.  If an error is encountered
// on the initial walk of a directory then that error is returned and no files
// are watched.  If the setup is successful then the returned channel will immediately
// receive an initial value of each file or directory being watched and the
// Update.WasAdded field will be set to true.
func Watch(done <-chan struct{}, path string, recurse bool, interval *time.Duration) (<-chan []Update, error) {
	root, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	var (
		infos   = make(map[string]os.FileInfo)
		updates = make(chan []Update)
		tick    = 2 * time.Second
	)

	if interval != nil {
		if *interval <= 0 {
			return nil, fmt.Errorf("Watch: interval may not be less than or equal to zero, given %d", *interval)
		}
		tick = *interval
	}

	werr := filepath.Walk(root, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		infos[path] = info

		if info.IsDir() && !recurse && path != root {
			return filepath.SkipDir
		}
		return nil
	})
	if werr != nil {
		return nil, werr
	}

	go (func() {
		idx, initial := 0, make([]Update, len(infos))
		for p, fi := range infos {
			initial[idx] = Update{AbsPath: p, Prev: fi, WasAdded: true}
			idx++
		}
		updates <- initial

		ticker := time.NewTicker(tick)
		defer ticker.Stop()
		defer close(updates)

		for {
			select {
			case <-ticker.C:
				var discovered []Update

				for path, prev := range infos {
					next, err := os.Stat(path)
					if err != nil {
						if os.IsNotExist(err) {
							delete(infos, path)
						}
						discovered = append(discovered, Update{
							AbsPath:    path,
							Prev:       prev,
							Error:      err,
							WasRemoved: true,
						})
					} else if prev.ModTime().Before(next.ModTime()) || prev.Size() != next.Size() {
						up := &Update{AbsPath: path, Prev: prev, Next: next}

						discovered = append(discovered, *up)
						infos[path] = next

						// check if new files have been added
						if next.IsDir() {
							up.Error = filepath.Walk(path, func(p string, i os.FileInfo, e error) error {
								if e != nil {
									return e
								} else if infos[p] == nil {
									infos[p] = i
									discovered = append(discovered, Update{
										AbsPath:  p,
										Prev:     i,
										WasAdded: true,
									})
									if i.IsDir() && !recurse {
										return filepath.SkipDir
									}
								}
								return nil
							})
						}
					}
				}
				if len(discovered) > 0 {
					updates <- discovered
				}
			case <-done:
				return
			}
		}
	})()

	return updates, nil
}
