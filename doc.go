// sks_spider is a tool to spider the PGP SKS keyserver mesh.
//
// A more introductory overview for usage should be in "README.md", as this
// code is geared for use as a program, not as a library.
//
// At present, the code is heavily geared towards providing one daemon,
// sks_stats_daemon.  This is a web-server which goes to a seed SKS server,
// grabs stats, and spiders out from there.
//
// The results are available over HTTP, with pages for humans and pages for
// automated retrieval.  The author successfully builds DNS zones using
// tools running out of cron which get their data from this server.
package sks_spider
