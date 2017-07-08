package par2

import (
	"bytes"
	"errors"
	"io"
)

type file struct {
	clientID               string
	mainPacket             *mainPacket
	fileDescriptionPackets map[fileID]fileDescriptionPacket
	unknownPackets         map[packetType][][]byte
}

func readFile(delegate DecoderDelegate, expectedSetID *recoverySetID, fileBytes []byte) (recoverySetID, file, error) {
	buf := bytes.NewBuffer(fileBytes)

	var setID recoverySetID
	var hasSetID bool
	if expectedSetID != nil {
		setID = *expectedSetID
		hasSetID = true
	}

	var foundPacket bool
	var clientID string
	var foundClientID bool
	var mainPacket *mainPacket
	fileDescriptionPackets := make(map[fileID]fileDescriptionPacket)
	unknownPackets := make(map[packetType][][]byte)
	for {
		packetSetID, packetType, body, err := readNextPacket(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			// TODO: Relax this check.
			return recoverySetID{}, file{}, err
		}
		if hasSetID {
			if packetSetID != setID {
				delegate.OnOtherPacketSkip(packetSetID, packetType, len(body))
				continue
			}
		} else {
			setID = packetSetID
			hasSetID = true
		}
		foundPacket = true
		switch packetType {
		case creatorPacketType:
			clientID = readCreatorPacket(body)
			delegate.OnCreatorPacketLoad(clientID)
			foundClientID = true

		case mainPacketType:
			// TODO: Handle duplicate main packets.
			mainPacketRead, err := readMainPacket(body)
			if err != nil {
				// TODO: Relax this check.
				return recoverySetID{}, file{}, err
			}

			mainPacket = &mainPacketRead
			delegate.OnMainPacketLoad(mainPacket.sliceByteCount, len(mainPacket.recoverySet), len(mainPacket.nonRecoverySet))

		case fileDescriptionPacketType:
			fileID, fileDescriptionPacket, err := readFileDescriptionPacket(body)
			if err != nil {
				// TODO: Relax this check.
				return recoverySetID{}, file{}, err
			}

			delegate.OnFileDescriptionPacketLoad(fileID, fileDescriptionPacket.filename, fileDescriptionPacket.byteCount)
			fileDescriptionPackets[fileID] = fileDescriptionPacket

		default:
			delegate.OnUnknownPacketLoad(packetType, len(body))
			unknownPackets[packetType] = append(unknownPackets[packetType], body)
		}
	}

	if !foundPacket {
		return recoverySetID{}, file{}, errors.New("no packets found")
	}

	if !foundClientID {
		return recoverySetID{}, file{}, errors.New("no creator packet found")
	}

	return setID, file{clientID, mainPacket, fileDescriptionPackets, unknownPackets}, nil
}
