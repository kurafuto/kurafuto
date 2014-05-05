Kurafuto (クラフト)
===================

Minecraft Classic load balancer, similar to nginx or BungeeCord (hopefully!).

## Topology

Ideally, this would act as a transparent proxy, acting more as a gateway/hub
server in a linked network of servers. Think nginx, but Minecraft Classic.

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

Due to the lack of a signaling packet that the balancer and servers can use to
indicate a server redirection, how servers _actually indicate_ the balancer's
connection should change from _Server A_ to _Server B_ is currently TBA.

Ideas include a custom [CPE](http://wiki.vg/Classic_Protocol_Extension) packet,
or some special chat message, which would be used as a sentinel of sorts.
Obviously, a CPE packet would be the preferred method, but we'll see.

When a client connects to the balancer, the balancer will make a connection on
the behalf of the client to one of the linked servers, and proxy packets back
and forth between the two.

## Proxy modes

Currently, there are two _"modes"_ of operation:

* __Parsing__ mode will parse packets out of the stream from the client, then
  forwards these to the server. The same is done for packets from the server.
  This allows the balancer to potentially inject extra packets or modify packets
  on-the-fly as they pass through.
* __Non-parsing__ mode simply reads everything out of the client stream, and
  pushes it straight on to the server. The same is done in the other direction.
  This mode is non-supportive of authentication and (currently) any plans for
  redirection (which is no fun!).

If you enable authentication, Kurafuto is forced into parsing mode. There is no
way around this.

## Authentication

This could work in a few ways:

* The servers (_A_, _B_, _C_, ...) handle their own authentication individually.
  They'd have to use a specific salt used by the balancer, however, otherwise
  clients might be able to connect to _A_, but not to _B_.
* The servers disable authentication (`verify-names=false`), and the balancer
  handles authentication at the edge.

The latter seems like the more reasonable solution currently, so that's what
will be implemented in pre-release versions.

Given that only valid connections will be coming from the edge balancer, and
the servers will have authentication disabled, it would make sense for the
servers to be configured to blacklist connections from anyone but the balancer.

## Heartbeats

The servers should disable their heartbeating (making them "private" servers),
and the edge balancer will heartbeat on the behalf of the servers.

This is still _TODO_, and exactly how it will work is currently TBA.

## Roadmap

Right now, the balancer _"works"_ (in quotation marks), but is definitely not
feature complete. There are going to be bugs, and it will probably crash a whole
lot, but that's how software works, right?

This is __definitely__ pre-release software, so it might go through sweeping
changes every few commits, and may: crash, burn, _accidentally_ lose packets,
eat your pets, and potentially burn down your house. You've been warned.

Things to work on:

* ~~Parse proxy mode~~ - works, minus authentication
* ~~Raw proxy mode~~ - seems functional, I haven't run into any issues.
* Handle `SIGHUP` to reload configuration (preferably without disconnecting clients)
* Heartbeats
    * Just ClassiCube?
* Authentication (requires parse mode so it's not hellish)
    * Just Classicube?
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
