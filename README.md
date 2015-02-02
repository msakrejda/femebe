# FEMEBE

FEMEBE (Front-End, Middle-End, Back-End) is a library for
introspection and manipulation of the [PostgreSQL wire protocol]
(http://www.postgresql.org/docs/9.2/static/protocol.html),
colloquially known as FEBE (Front-End/Back-End).

The library is in development and breaking changes to its API
will still happen.

FEMEBE works with all currently-supported Postgres versions.

## Overview

The Postgres protocol abstracts the low-level TCP communication
between a client and server into a bidrectional stream of sequential,
synchronous messages.

FEMEBE provides an interface to that abstraction, so any Go tool that
needs to hook into the protocol can build against a normal package API
rather than having to parse the raw protocol at the TCP level.

There are five basic ways to use FEMEBE:

 * Stub out or implement a Postgres client (the former is probably
   more sensibly done by actually *being* a client--e.g., using
   [pq](https://github.com/lib/pq)--but this may be useful for more
   esoteric use cases).
 * Stub out or implement (!) a Postges-compatible server
 * Postgres switchboard (route connections and then step out of the
   way)
 * Listen to the protocol (e.g., a protocol traffic viewer)
 * Manipulate the protocol (e.g., dynamically enrich query results
   with data from other sources)

## Contributing

FEMEBE can use help in a number of areas:

 * Bug reports, especially with minimal, reproducible test cases
 * Performance benchmarks and improvements
 * API improvements so we can confidently freeze the interface,
   especially for:
   * Composable (but still performant and simple) `Router`s
   * Custom error types rather than blind propagation
 * Add server TLS support (TLS for the client piece exists)
 * Support parsing and formatting remaining message types
 * Utility functions for data type management
 * Documentation (especially sample code)
 * Other ideas

To contribute, please open an issue.
