sks\_spider
===========

Tool to spider the PGP SKS keyserver mesh.

[![Build Status](https://api.travis-ci.org/philpennock/sks_spider.png?branch=main)](https://travis-ci.org/philpennock/sks\_spider)

This code-base is horrible; it was predominantly written in a weekend, porting
from some very organic Python.  Do not use this as an example of how to do
things in Golang.

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
unperturbed by the migration.

The original version was written in Python as a WSGI and grew organically.
This version is written in Golang (the Go programming language) and makes
fairly decent use of Go's concurrency features.  It uses well under a fifth
the total RAM, something similarly smaller in RSS, uses less CPU (when busy,
10% of an ancient CPU instead of all of one; when
"idle" is not sitting at the top of top(1) output, using fractionally more
CPU than a real idle process) and is _significantly_ more responsive.  These
improvements are in part because of Golang and in very large part because of
the ugliness of the old code.  Python's good, I'm bad.

All the production serving interface features have now been copied across;
all that's left are some admin hooks which aren't really applicable (eg,
a list of Python threads for introspection).
Those other features should not significantly impact resource consumption.


To-Do
-----

* Preserve more errors for the front-page?
* Look over the admin interfaces, probably want `/rescanz` back
* If add rescanz, need locking around spider starting; can preserve spider
  handle while at it, and make it possible to, eg kill an existing scan using
  a random nonce to authenticate, where the nonce has to be retrieved from
  the logfile.


Building
--------

You will need development headers installed:

* Debian/Ubuntu etc: `apt-get install libxml2-dev`
* MacOS: `brew install libxml2`

To fetch the code, all dependencies, updating them, and install the command,
then run:

    go get -u -v github.com/philpennock/sks_spider/...
    # optionally:
    cd ~/go/src/github.com/philpennock/sks_spider
    make

You don't _have_ to use `make`, but it does embed a version string into the
binary which can be operationally useful.  Note that GNU make should be used
(or any other make implementation which handles `GNUmakefile` adequately).

If you encounter problems, look at the `.travis.yml` file which is used
for running the Travis Continuous Integration tests:
<https://travis-ci.org/philpennock/sks_spider>.
That assumes some other prep steps run automatically by Travis, but the test
log should show everything in context.


Running
-------

You can see the accepted parameters with the `-help` flag:

    sks_stats_daemon -help

You might run, as an unprivileged user:

    sks_stats_daemon -log-file /var/log/sks-stats.log

Note that this tool does not self-detach from the terminal: I prefer to leave
it where a supervising agent tool can easily watch it.  If you want it to
detach, then your OS should have available a wrapper command which will handle
that for you.

The log-file will need to be exist and be writeable by that unprivileged
user (or be in a directory which that user can create new files in).

Note that the logging does not currently log all HTTP requests; that's the
responsibility of the front-end (for now?).  Actually, the logging isn't
production-grade.  It "logs", but that doesn't mean the logs have proven
themselves adequate at crunch time.

The horrible HTML templates (translated directly from my horrible Python
ones ... I'm _definitely_ not a UI designer) expect a style-sheet and a
`favicon.ico` to be provided as part of the namespace, they're not served
by this daemon.

Yes, this is a toy program.  It's a useful toy, but definitely not a
shipping product.


My start-up script (OS-specific, not included) `touch`es and `chown`s
the log-file before starting the program.  It then runs, as the same
run-time user as is used for `sks` itself (for my convenience in user
management):

    sks_stats_daemon -log-file /var/log/sks-stats.log \
      -json-persist /var/sks/stats-persist.json \
      -started-file /var/sks/stats.started

The `-json-persist` flag causes `sks_stats_daemon` to register a handler
for `SIGUSR1`; receipt of that signal causes the current mesh to be written to
the named file (removing any previous content), before exiting.

The start-up script takes a `quickrestart` argument, which sends `SIGUSR1`,
waits for the process to disappear, then starts `sks_stats_daemon` once more.
It then waits for the `-started-file` flag-file to appear, then removes it
and exits.


nginx configuration
-------------------

It's as simple as:

    location /sks-peers {
        proxy_pass          http://127.0.0.1:8001;
        proxy_set_header    X-Real-IP $remote_addr;
    }

In fact, you don't even need the X-Real-IP pass-through, but set it up now
and it'll be easier to deal with a future change which logs the origin IP.

Note especially that, as suggested in the Overview above, we're only passing
through the `/sks-peers` part of the namespace; this avoids exposing the
`/debug` hierarchy, amongst others.


License
-------

Apache 2.0.

Most people are nice and sane and in a world without subversion and lawyers,
this next bit wouldn't be necessary.  It's butt-covering, that's all.

If you send me a patch or a pull request, then by default:
 * I will add you to a CONTRIBUTORS file
 * You are assumed to be implicitly granting a license to me for your work to
   be distributed under the same license, as part of a larger work
 * You are assumed to have the authority to submit the modification
   under these terms and are implicitly testifying to this by making the
   submission.

In other words: please don't be a jackass, contributions are expected to
contribute towards the codebase, not take away.  Thanks.

That's about it.  
-Phil

Copyright 2012,2013,2016 Phil Pennock.
