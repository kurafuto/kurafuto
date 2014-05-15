Kurafuto (クラフト)
===================

Minecraft Classic load balancer, similar to nginx or BungeeCord (hopefully!).

This is __definitely__ pre-release software, so it will probably go through sweeping
changes every few commits, and may: crash, _"accidentally"_ lose packets, eat
your pets, or burn down your house if you look at it wrong. You've been warned.

What I'm saying is, __don't use this in production yet.__

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

******

Due to the lack of a signaling packet that the balancer and servers can use to
indicate a server redirection, how servers _actually indicate_ the balancer's
connection should change from _Server A_ to _Server B_ is currently TBA.

Ideas include a custom [CPE](http://wiki.vg/Classic_Protocol_Extension) packet,
some special chat message used as a sentinel of sorts, or a user-triggered command.
Obviously, a CPE packet would be the preferred method, but that'd require lots
of effort, and who wants that..?

When a client connects to the balancer, the balancer will make a connection on
the behalf of the client to one of the linked servers, and proxy packets back
and forth between the two.

## Proxying

Kurafuto makes use of [Kyubu](https://github.com/sysr-q/kyubu) to parse packets
out of the client and server streams. This allows the balancer to do things like
inject, drop, or modify packets on-the-fly as they pass through.

This does mean, however, that any unrecognised packets will be an issue - if
there's a custom packet you want to recognise, be sure to register it with Kyubu
(which is quite simple), and Kurafuto will pass it through just fine.

## Authentication

Authentication (if enabled: `"authentication": true`) is handled at the edge by
the balancer. This means that servers will have to disable their authentication
and throttling limitations.

Given that only valid connections will be coming from the edge balancer, and
the servers will have authentication disabled, it would make sense for the
servers to be configured to blacklist connections from anyone but the balancer.

## Heartbeats

The servers should disable their heartbeating (making them "private"/"hidden"),
and the edge balancer will heartbeat on the behalf of the servers.

This is still _TODO_, and exactly how it will work is currently TBA.

## Roadmap

Things to work on:

* ~~Parse proxy mode~~ - mostly works, there are some odd issues with LevelDataChunk packets.
* Handle `SIGHUP` to reload configuration (preferably without disconnecting clients)
* Heartbeats
    * Just ClassiCube?
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
