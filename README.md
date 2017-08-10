filewatch
=========

A Go library to watch for changes of a file or a directory and its contents.

Overview
--------

This library allows one to watch for file changes in a directory with
the option of watching sub-directories recursively or to just watch a
single file.  See `filewatch.go` for full documentation and usage.
Also, `demo/demo.go` contains a simple monitoring program illustrating
the usage of the library.

Known Issues
------------

A file change is detected when the scanned modification time differs
from a previously taken modification time or the size of a file has
changed.  However, even though time package in Go's standard library
allows specifying the nanosecond in the `time.Time` type, this
nanosecond isn't recorded when writing a change to a file (most likely
depends on the underlying OS).  Thus, if a change or a series occurs
to a file or directory within a single second and the size hasn't
changed from the last recorded file change then non change will be
detected.  In practice, this issue may be pretty rare depending on how
one uses the library but important to note.

License
-------

Zlib
