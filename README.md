# Intro
Package fsnotifydeep extends fsnotify to make it work recursively. During an Add,
all subdirectories are added as well. For a Remove, the same. As the contents
of the monitored directories changes, fsnotify is updated accordingly.

In addition, fsnotifydeep adds a Filter feature to Watchers, allowing the client
to specify a function literal that determines if an event should appear on the
Events channel or be quietly dropped.

The primary downside of this package is a performance hit. The exact amount
will depend on the size of the filesystem under monitor.

# Docs
As with all Go packages, documentation can be found on
[godoc.org](https://godoc.org/github.com/traherom/fsnotifydeep). The API
is compatible with [fsnotify's](https://github.com/go-fsnotify/fsnotify) v1 API.
Recursive watching occurs automatically and a filter can be added easily.

# Install
     $ go get github.com/traherom/fsnotifydeep
