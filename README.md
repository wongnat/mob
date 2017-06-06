<pre>
              ___.
  _____   ____\_ |__
 /     \ /  _ \| __ \
|  Y Y  (  <_> ) \_\ \
|__|_|  /\____/|___  /
      \/           \/
</pre>
# Internet Radio

## Description

A semi peer-to-peer internet radio application.
Clients can connect to a tracker and enqueue songs
to be played to all clients. When clients join a tracker, their
songs in the `songs` folder are visible to all other clients.
If a song is enqueued and a client does not have it locally, one of
its peers will stream it to them.

Notes:
* Only mp3 is supported
* Only works over local NAT for now

## Usage

#### Build
`make build`

#### Run the client
`
cd bin
./client
`

Alternatively,

`
cd client
go run client.go
`

#### Run the tracker
`
cd bin
./tracker <port>
`

Alternatively,

`
cd tracker
go run tracker.go <port>
`

## Dependencies

* [go-SDL2](https://github.com/veandco/go-sdl2) - golang SDL2 bindings to play mp3s
* [mp3](https://github.com/tcolgate/mp3) - golang mp3 library to parse mp3 frames
* [rpc2](https://github.com/cenkalti/rpc2) - golang rpc library for communication between clients and trackers

## Platforms

Linux, Mac OSX, Windows

## TODO

* Write tests
* Fix relative paths to resources
* Allow clients to join and play audio mid-stream
* Improve synchronization of audio among clients
* Remove tracker and make fully peer-to-peer
* Support other audio file types

## License

MIT License Copyright (c) 2017 Nathan Wong, Sudharsan Prabu, Tariq Amireh
