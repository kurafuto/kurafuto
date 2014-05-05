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

## Roadmap

Right now, the balancer _"works"_ (in quotation marks), but is definitely not
feature complete. There are going to be bugs, and it will probably crash a whole
lot, but that's how software works, right?

Things to work on:

* ~~Parse proxy mode~~ (works, minus authentication)
* Raw proxy mode
* Handle `SIGHUP` to reload configuration (preferably without disconnecting clients)
* Heartbeats
* Authentication (requires parse mode so it's not hellish)
* Handling redirection signals
    * There are a few ways this could work, varying from decent to quite hacky,
      so we'll have to try out a few and see how they work. Will we buffer some
      packets (namely, `Identification`, `ExtInfo`, `ExtEntry`) and push those
      on to the new server?
