sks\_spider contrib
===================

This directory contains auxilliary tools/scripts/miscellanea not needed
for sks\_spider itself.

`update_sks_zone`
-----------------

This Perl script is what I (Phil) use to generate/update a DNS zonefile,
retrieving data from sks\_spider.

Nonstandard Perl dependencies: `JSON::XS`, `LWP::Simple`, `Socket6`

Caveats:

1. default file-paths, all of which can be overriden on the command-line,
   are very specific to one system.
2. I haven't written Perl as a first choice in a few years, I haven't kept up
   with current best practices and mostly this isn't recent code.
3. Doesn't trigger zone reload; that's handled by another tool, not included,
   that is even more system-specific (`dnssync_update_domains`).
4. Geographic regions and their TLDs are hard-coded

That said, this may prove useful.

*Please Note*: the community is not well served by many competing PGP keyserver
pools, all claiming to be authoritative, coming and going and leaving a trail
of now-broken `gpg.conf` files in the wake of their devastation.  There is one
common set of dynamic pools, maintained by Kristian Fiskerstrand at
<http://www.sks-keyservers.net/>, which users should be encouraged to use.  As
can be seen from the default zone-name in this script, I use an obnoxiously
long name to discourage use.

For me, the value of this zone has been two-fold:

1. Educational, for me
2. Friendly competition as Kristian and I both improved the functionality and
   robustness of our keyserver DNS pool generators.

Note that point 2 did not require that anyone other than Kristian try using my
pool.  :)  There's more than one type of competition.
