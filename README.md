sks\_spider
===========

Tool to spider the PGP SKS keyserver mesh.

Overview
--------

If you don't know what PGP is or anything about the PGP keyservers, then
this tool is not for you.  Otherwise, read on.

This is a package which produces one binary, `sks_stats_daemon`.  This is a
web-server which goes to a seed SKS server, grabs stats, and spiders out
from there.

The resulting daemon should be set behind a front-end web-server such as
nginx, with the `/sks-peers` location dispatched to it.  If you run this
daemon listening on a publicly reachable port, or dispatch more of the URI
namespace to the daemon, you may have issues, as administrative URIs can
live outside of that prefix.

As well as a stats overview page, there is also an interface to grab lists
of IPs meeting various serving criteria; I use that to build DNS zones
automatically, from cron, as a client of this service.  The client was
unperturbed by the migration, cleanly dropping the sub-zones not currently
supported.

The original version was written in Python as a WSGI and grew organically.
This version is written in Golang (the Go programming language) and makes
fairly decent use of Go's concurrency features.  It uses well under a fifth
the total RAM, something similarly smaller in RSS, uses less CPU (when busy,
10% of an ancient CPU instead of all of one; when
"idle" is not sitting at the top of top_(1)_ output, using fractionally more
CPU than a real idle process) and is _significantly_ more responsive.  These
improvements are in part because of Golang and in very large part because of
the ugliness of the old code.  Python's good, I'm bad.

At present, not all the features have been copied across, but what's left is
less fundamental and can be pulled across at my leisure.  Those other
features should not significantly impact resource consumption.


To-Do
-----

* Preserve more errors for the front-page?
* `.dot` file generation
* Look over the admin interfaces, probably want `/rescanz` back
* If add rescanz, need locking around spider starting; can preserve spider
  handle while at it, and make it possible to, eg kill an existing scan using
  a random nonce to authenticate, where the nonce has to be retrieved from
  the logfile.

Building
--------

For the most part, `go get` should just work.

The exception is the btree support from https://github.com/runningwild/go-btree
which is very nice, and written using generics, with the `gotgo`
pre-processor needed to emit a .go file.

Grab https://github.com/droundy/gotgo and put some go1 `// +build ignore`
magic into a couple of the benchmark files, and you'll be able to build
the `gotgo` and `gotimports` commands.

In the gotgo directory, run:

    gotgo -o btree.go btree.got string

There's probably a better way to sort out a namespace hierarchy which don't
expect only one instantiation of the generic, but I was grabbing a btree
library in passing and didn't investigate fully.

After that, the btree import will work and the code should build with:

    go build github.com/syscomet/sks_spider/sks_stats_daemon.go

(Of course, I write this before pushing to github, so maybe not ...)

Running
-------

I run, as an unprivileged user:

    sks_stats_daemon -log-file /var/log/sks-stats.log

The start-up script (OS-specific) touches and chowns the log-file before
starting the program.  Note that this tool does not self-detach from the
terminal: I prefer to leave it where a supervising agent tool can easily
watch it.

Note that the logging does not currently log all HTTP requests; that's the
responsibility of the front-end (for now?).  Actually, the logging isn't
production-grade.  It "logs", but that doesn't mean the logs have proven
themselves adequate at crunch time.

The horrible HTML templates (translated directly from my horrible Python
ones ... I'm *definitely* not a UI designer) expect a style-sheet and a
`favicon.ico` to be provided as part of the namespace, they're not served
by this daemon.

Yes, this is a toy program.  It's a useful toy, but definitely not a
shipping product.


nginx configuration
-------------------

It's as simple as:

    location /sks-peers {
        proxy_pass          http://127.0.0.1:8001;
        proxy_set_header    X-Real-IP $remote_addr;
    }

In fact, you don't even need the X-Real-IP pass-through, but set it up now
and it'll be easier to deal with a future change which logs the origin IP.


License
-------

Apache 2.0.


That's about it.
-Phil
