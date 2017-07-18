package par2

func byteToUint16LEArray(bs []byte) []uint16 {
	u16s := make([]uint16, len(bs)/2)
	for i := 0; i < len(u16s); i++ {
		u16s[i] = uint16(bs[2*i]) + uint16(bs[2*i+1])<<8
	}
	return u16s
}

func uint16LEToByteArray(u16s []uint16) []byte {
	bs := make([]byte, 2*len(u16s))
	for i := 0; i < len(u16s); i++ {
		bs[2*i] = byte(u16s[i])
		bs[2*i+1] = byte(u16s[i] >> 8)
	}
	return bs
}
