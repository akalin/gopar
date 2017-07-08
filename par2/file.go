package par2

import (
	"bytes"
	"errors"
	"io"
	"reflect"
)

type file struct {
	clientID               string
	mainPacket             *mainPacket
	fileDescriptionPackets map[fileID]fileDescriptionPacket
	ifscPackets            map[fileID]ifscPacket
	recoveryPackets        map[exponent]recoveryPacket
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
	ifscPackets := make(map[fileID]ifscPacket)
	recoveryPackets := make(map[exponent]recoveryPacket)
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

		case ifscPacketType:
			fileID, ifscPacket, err := readIFSCPacket(body)
			if err != nil {
				// TODO: Relax this check.
				return recoverySetID{}, file{}, err
			}

			delegate.OnIFSCPacketLoad(fileID)
			ifscPackets[fileID] = ifscPacket

		case recoveryPacketType:
			exponent, recoveryPacket, err := readRecoveryPacket(body)
			if err != nil {
				// TODO: Relax this check.
				return recoverySetID{}, file{}, err
			}

			delegate.OnRecoveryPacketLoad(uint16(exponent), 2*len(recoveryPacket.data))
			if existingPacket, ok := recoveryPackets[exponent]; ok {
				if !reflect.DeepEqual(existingPacket, recoveryPacket) {
					return recoverySetID{}, file{}, errors.New("recovery packet with duplicate exponent but differing contents")
				}
			}
			recoveryPackets[exponent] = recoveryPacket

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

	return setID, file{clientID, mainPacket, fileDescriptionPackets, ifscPackets, recoveryPackets, unknownPackets}, nil
}
