// All functions for our protocol go here:
package proto

import (
    "github.com/tcolgate/mp3"
)

type Client_Init_Packet struct {
    Songs string
}

type Mp3_Frame_Packet struct {
    Seqnum uint64
    Mp3_frame []byte
}

// Other pacekt types
