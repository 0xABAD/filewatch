package filewatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatch(t *testing.T) {
	root := filepath.Join(os.TempDir(), "root")
	sub := filepath.Join(root, "sub")

	if err := os.MkdirAll(sub, os.ModeDir|os.ModePerm); err != nil {
		t.Fatal("Can't create required directories --", err)
	}
	defer os.RemoveAll(root)

	temp1 := filepath.Join(root, "temp1")
	temp2 := filepath.Join(sub, "temp2")

	file1, err := os.Create(temp1)
	if err != nil {
		t.Fatal("Can't create file temp1 --", err)
	}
	defer file1.Close()

	file2, err := os.Create(temp2)
	if err != nil {
		t.Fatal("Can't create file temp2 --", err)
	}
	defer file2.Close()

	t.Run("Fails with bad interval", func(t *testing.T) {
		interval := time.Duration(0)
		done := make(chan struct{})
		defer close(done)

		_, err := Watch(done, root, false, &interval)
		if err == nil {
			t.Fatal("Didn't fail with interval of 0 or less")
		}
	})

	interval := 10*time.Millisecond
	getUpdates := func(updates <-chan []Update) []Update {
		select {
		case u := <-updates:
			return u
		case <-time.After(2 * interval):
			return nil
		}
	}

	t.Run("Sends initial update", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)

		updates, err := Watch(done, root, true, &interval)
		if err != nil {
			t.Fatal("Couldn't establish a valid watch --", err)
		}

		count, initial := 0, getUpdates(updates)
		if initial == nil {
			t.Fatal("Did not receive an update by the timeout")
		}

		for _, u := range initial {
			if !u.WasAdded {
				t.Error("Initial update didn't have WasAdded set to true")
			}
			switch u.AbsPath {
			case root:
				fallthrough
			case sub:
				if !u.Prev.IsDir() {
					t.Error(u.AbsPath, "is not a directory")
				}
			case temp1:
			case temp2:
			default:
				t.Error("Initial update for", u.AbsPath, "didn't match any of test paths")
			}
			count++
		}
		if count != 4 {
			t.Fatal("There were not 4 updates in the initial update but", count, "instead")
		}
	})

	t.Run("Sends update on file change", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)

		updates, err := Watch(done, root, true, &interval)
		if err != nil {
			t.Fatal("Couldn't establish a valid watch --", err)
		}

		if getUpdates(updates) == nil { // skip initial updates
			t.Fatal("Did not receive initial update by the timeout")
		}

		const foo = "foo"
		n, err := file1.WriteString(foo)
		if err != nil {
			t.Fatal("Could not write a byte to file1 --", err)
		} else if n != 3 {
			t.Fatal("Did not write 'foo' to file1")
		} else if err := file1.Sync(); err != nil {
			t.Fatal("Failed to Sync() file1")
		}
		defer func() {
			file1.Truncate(0)
			file1.Sync()
		}()

		count, next := 0, getUpdates(updates)
		if next == nil {
			t.Fatal("Did not receive update by the timeout")
		}

		for _, u := range next {
			switch u.AbsPath {
			case temp1:
				if u.Next == nil {
					t.Error("Update.Next does not have FileInfo")
				} else if u.Next.Size() != int64(len(foo)) {
					t.Error("Did not write", len(foo), "bytes to temp1")
				}
			default:
				t.Error("Received an update for", u.AbsPath, "when it shouldn't have")
			}
			count++
		}
		if count != 1 {
			t.Fatal("There was not 1 update in but", count, "instead")
		}
	})

	interval = 10 * time.Millisecond

	temp3 := filepath.Join(sub, "temp3")
	var file3 *os.File

	// This also implicity checks updates are checked when recurse is true
	t.Run("Sends update on file addition", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)

		updates, err := Watch(done, root, true, &interval)
		if err != nil {
			t.Fatal("Couldn't establish a valid watch --", err)
		}

		if getUpdates(updates) == nil { // skip initial updates
			t.Fatal("Did not receive initial update by the timeout")
		}

		var cerr error
		file3, cerr = os.Create(temp3)
		if cerr != nil {
			t.Fatal("Could't create file temp3 --", cerr)
		}

		count, next := 0, getUpdates(updates)
		if next == nil {
			t.Fatal("Did not receive update by the timeout")
		}

		for _, u := range next {
			switch u.AbsPath {
			case sub:
			case temp3:
				if !u.WasAdded {
					t.Error(u.Prev.Name(), "was not marked as added")
				}
			default:
				t.Error("Received an update for", u.AbsPath, "when it shouldn't have")
			}
			count++
		}
		if count != 2 {
			t.Fatal("There were not 2 updates in but", count, "instead (no file added)")
		}
	})

	file3.Close()

	t.Run("Sends update on file deletion", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)

		updates, err := Watch(done, root, true, &interval)
		if err != nil {
			t.Fatal("Couldn't establish a valid watch --", err)
		}

		if getUpdates(updates) == nil { // skip initial updates
			t.Fatal("Did not receive initial update by the timeout")
		}

		if err := os.Remove(temp3); err != nil {
			t.Fatal("Couldn't delete file temp3 --", err)
		}

		count, next := 0, getUpdates(updates)
		if next == nil {
			t.Fatal("Did not receive update by the timeout")
		}

		for _, u := range next {
			switch u.AbsPath {
			case sub:
			case temp3:
				if !u.WasRemoved {
					t.Error(u.Prev.Name(), "was not marked as removed")
				}
			default:
				t.Error("Received an update for", u.AbsPath, "when it shouldn't have")
			}
			count++
		}
		if count != 2 {
			t.Fatal("There were not 2 updates in but", count, "instead (no file removed)")
		}
	})
}
