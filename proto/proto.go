package proto

type Client_Init_Packet struct {
    Songs string
}

type Mp3_Frame_Packet struct {
    Seqnum uint64
    Mp3_frame []byte
}

// More packet types:


// Protocol functions:
