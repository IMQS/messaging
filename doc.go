/*
Package messaging sends SMS notifications to large batches of numbers

CR: It's nice to have a 'doc.go' file like this, for providing a place
where one can write overview documentation of a package.

For a nice manifestation of this, see these two examples:

    https://godoc.org/github.com/IMQS/authaus
    https://godoc.org/github.com/IMQS/router-core/router

Since those packages are open source, we simply tell "godoc.org" about
them once, and thereafter it automatically updates them from github.

------------------------------------------------------------------
Overall design comment:

A service like this should have a root object. So "StartServer"
shouldn't be a global function. It should be a method called on an
instance of a type such as "type MessagingServer struct".
That type must be the root type that holds all the mutable state
of the service, as well as the config. As the service stands right now,
it has close to zero mutable state, which is great, but we must still
stick to that design, so that we have a root object in which to place
all service state. In particular, things that do need to go inside
MessagingServer are the 'quit' channel for the interval ticker,
the logger instance (of github.com/IMQS/log), Config, DB, DBCon.
Although, I don't think there should be two copies of DBCon. The one
inside the Config should suffice.

I'm not suggesting that all functions inside this package become
members of MessagingServer, but the core entry points such as
StartServer, should be.

I find the design of the Message and Response data structures confusing.
I think it either needs a solid explanation, or perhaps a slightly
tweaked design to make it more obvious how it all fits together.

*/
package messaging
