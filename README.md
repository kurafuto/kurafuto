Kurafuto (クラフト)
===================

Minecraft Classic load balancer, similar to nginx or BungeeCord (hopefully!).

This is __definitely__ pre-release software, so it will probably go through sweeping
changes every few commits, and may: crash, _"accidentally"_ lose packets, eat
your pets, or burn down your house if you look at it wrong. You've been warned.

What I'm saying is, __don't use this in production yet.__

_NB!_ The phrases "balancer", "proxy" and "Kurafuto" are used somewhat interchangeably
in this document. All it means is the box between clients and backend servers.

## Usage

There are a few steps to using Kurafuto in its current state:

```
# Install Kurafuto
$ go get github.com/sysr-q/kurafuto
$ go install github.com/sysr-q/kurafuto

# Set up where you're "running" Kurafuto from
$ mkdir /path/to/kura
$ cd /path/to/kura
$ cp $GOPATH/github.com/sysr-q/kurafuto/kurafuto.json .

# Modify the configuration to your liking & run!
$ vim kurafuto.json
$ ./kurafuto -config="$(pwd)/kurafuto.json"
```

## Topology

Ideally, this would act as a transparent proxy, acting more as a gateway/hub
server in a linked network of servers. Think an IRC network, in terms of hub/leaf
server links.

```
                               [ Server A ]
                            ____/
        [ Client 1 ]       /         [ Server B ]
                  \       /       ____/
 [ Client 2 ] -- [ Kurafuto ] ---`
                  /       \_____
        [ Client 3 ]            \
                                 [ Server C ]

Clients: 1, 2, 3                 Servers: A, B, C
```

******

As there is no native signal for a server to indicate that a client (or a balancer
masquerading as a client) should jump server, Kurafuto has to make use of some
imperfect workaround solutions in the mean time.

Implementation ideas have included:

* A client-side command, which Kurafuto intercepts, allowing the client to
  "force" a redirection. `:kura jump ServerA` , for example. _This is what is
  currently implemented._
* Some horrifying sentinel packet between the server and Kurafuto, in the form
  of `"\x00REDIRECT\xFFfoo.example.com:31337\xFF"`.
* A custom [CPE](http://wiki.vg/Classic_Protocol_Extension) packet, which the
  server could send, indicating where to jump to. This would be nice (and would
  allow uses outside of Kurafuto), but getting a new packet into the spec is _hard_.
  This would also require extra server work (to get around authentication when
  not trusting a balancer), and this is a lot of hard work.

## Proxying

When a client connects to the balancer (Kurafuto), the balancer will make a
connection on the behalf of the client to one of the linked servers, and proxy
packets back and forth between the two.

Kurafuto makes use of [Kyubu](https://github.com/sysr-q/kyubu) to parse packets
out of the client and server streams. This allows the balancer to do things like
inject, drop, or modify packets on-the-fly as they pass through.

Use of a custom parser, rather than a dumb TCP proxy does mean, however, that any
unrecognised packets will be an issue - if there's a custom packet you want to
recognise, be sure to register it with Kyubu (which is documented in Kyubu's repo,
and quite simple), and Kurafuto will pass it through just fine.

Note, though, that the packet id `0xff` is given special meaning: it's used to
register packet handlers which listen for _any_ packet. This might be an issue
if a future packet uses that id.

## Authentication

Authentication (if enabled: `"authentication": true`) is handled at the edge by
the balancer. This means that servers will have to disable their authentication
and any throttling limitations.

If authentication _is_ enabled, this means that only valid connections will be
coming from the edge balancer, and the servers will have authentication disabled,
it would make sense for the servers to be configured to blacklist connections
from anyone but the balancer.

## Heartbeats

Backend servers should either _disable_ their heartbeats, or if this isn't possible,
set themselves to _"private"_, ensuring that there aren't servers in the public
listings which shouldn't be present.

Kurafuto will be able to make heartbeats on the behalf of the servers, but the
exact specifics of this (how the MOTD, etc. are set) is still TBA.

## Roadmap (haphazard)

Things to work on:

* ~~Parse proxy mode~~ - works, and no longer do LevelDataChunk's cause issues.
* Handle `SIGHUP` to reload configuration (preferably without disconnecting clients)
* ~~Handle `SIGINT` and `SIGTERM` to gracefully shut down (kicking clients).~~
* Heartbeats
    * Just ClassiCube? Presumably.
* ~~Authentication (requires parse mode so it's not hellish)~~
    * ClassiCube authentication is supported.
* Forwarding on the "real" IP in an `X-Forwarded-For` manner. Might try to get
  an extra packet into the CPE spec.
* There's a slight delay when users connect where the balancer dials to the hub
  server. What can we do about that? Keep a small "pool" of connections that we
  refresh periodically? Not sure whether that's viable.
* Handling redirection signals
    * There are a few ways this could work, varying from decent to quite hacky,
      so we'll have to try out a few and see how they work. Will we buffer some
      packets (namely, `Identification`, `ExtInfo`, `ExtEntry`) and push those
      on to the new server?
    * A custom client command seems like a decent idea. Easy to intercept as well.
* Perhaps a way to easily register custom packets in the config file, so that
  while we don't _care_ about the packets, they'll still be passed through
  without the need to recompile anything.
* Allow multiple Kurafuto servers to mesh link sideways, allowing extra crazy
  setups, and load balancing? Maybe if there's a good reason, we'll see.
* Daemonizing (hard in Go), multiplexing log files, storing a pidfile, all the
  usual stuff you'd expect a long-running server process to do.
* Add extra debugging information, tidy up existing information, and ensure
  that (in the case of bugs), it's all easily accessible to server admins.
