gta: go test auto
================
`gta` builds a list of "dirty" (changed) packages from master, using git.

`gta` works by implementing a various set of interfaces, namely the `Differ` and `Packager` interfaces.
