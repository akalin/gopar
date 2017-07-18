package par2

var creatorPacketType = packetType{'P', 'A', 'R', ' ', '2', '.', '0', '\x00', 'C', 'r', 'e', 'a', 't', 'o', 'r'}

func readCreatorPacket(body []byte) string {
	return decodeNullPaddedASCIIString(body)
}

func writeCreatorPacket(clientID string) ([]byte, error) {
	return encodeASCIIString(clientID)
}
