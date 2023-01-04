# Message Queue example

This example demonstrates message queue pattern using Redis implementation of IMessageBus in `yaaf-common` package.
The example initializes two publishers writing messages to the queue and two subscribers pulling these messages

### Message Queue pattern
In a message queue, many publishers can publish (send) a message to a queue and many subscribers can pull messages from a queue.
A message is processed *only once*, it is picked up by a subscriber and removed from the queue.

```mermaid
flowchart LR
   p1(Publisher 1) --> q1[(water queue)]
   p2(Publisher 2) --> q1[(water queue)]
   q1[(water queue)] --> c1(Consumer 1)
   q1[(water queue)] --> c2(Consumer 2)
```

To run this example:

```shell
go run .
```