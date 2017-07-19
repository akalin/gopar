package par2

import (
	"bytes"
	"crypto/md5"
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

type noPacketsFoundError struct{}

func (noPacketsFoundError) Error() string {
	return "no packets found"
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
			//
			// TODO: Warn if the recovery set ID computed
			// from the main packet doesn't equal setID.
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
		return recoverySetID{}, file{}, noPacketsFoundError{}
	}

	if !foundClientID {
		return recoverySetID{}, file{}, errors.New("no creator packet found")
	}

	return setID, file{clientID, mainPacket, fileDescriptionPackets, ifscPackets, recoveryPackets, unknownPackets}, nil
}

func padPacketBytes(packetBytes []byte) []byte {
	if len(packetBytes)%4 == 0 {
		return packetBytes
	}
	return append(packetBytes, make([]byte, 4-len(packetBytes)%4)...)
}

func writeFile(file file) (recoverySetID, []byte, error) {
	if len(file.clientID) == 0 {
		return recoverySetID{}, nil, errors.New("empty client ID")
	}

	if file.mainPacket == nil {
		return recoverySetID{}, nil, errors.New("no main packet")
	}

	buf := bytes.NewBuffer(nil)

	// TODO: Sanity-check packets against file.mainPacket.

	mainPacketBytes, err := writeMainPacket(*file.mainPacket)
	if err != nil {
		return recoverySetID{}, nil, err
	}
	paddedMainPacketBytes := padPacketBytes(mainPacketBytes)
	setID := recoverySetID(md5.Sum(paddedMainPacketBytes))

	creatorPacketBytes, err := writeCreatorPacket(file.clientID)
	if err != nil {
		return recoverySetID{}, nil, err
	}
	err = writeNextPacket(buf, setID, creatorPacketType, padPacketBytes(creatorPacketBytes))
	if err != nil {
		return recoverySetID{}, nil, err
	}

	err = writeNextPacket(buf, setID, mainPacketType, paddedMainPacketBytes)
	if err != nil {
		return recoverySetID{}, nil, err
	}

	for _, fileID := range append(file.mainPacket.recoverySet, file.mainPacket.nonRecoverySet...) {
		fileDescriptionPacket, ok := file.fileDescriptionPackets[fileID]
		if !ok {
			return recoverySetID{}, nil, errors.New("could not find file description packet")
		}

		fileDescriptionPacketBytes, err := writeFileDescriptionPacket(fileID, fileDescriptionPacket)
		if err != nil {
			return recoverySetID{}, nil, err
		}
		err = writeNextPacket(buf, setID, fileDescriptionPacketType, padPacketBytes(fileDescriptionPacketBytes))
		if err != nil {
			return recoverySetID{}, nil, err
		}

		ifscPacket, ok := file.ifscPackets[fileID]
		if !ok {
			return recoverySetID{}, nil, errors.New("could not find input file slice checksum packet")
		}

		ifscPacketBytes, err := writeIFSCPacket(fileID, ifscPacket)
		if err != nil {
			return recoverySetID{}, nil, err
		}
		err = writeNextPacket(buf, setID, ifscPacketType, padPacketBytes(ifscPacketBytes))
		if err != nil {
			return recoverySetID{}, nil, err
		}
	}

	// TODO: Sort exponents before iterating.
	for exp, packet := range file.recoveryPackets {
		recoveryPacketBytes, err := writeRecoveryPacket(exp, packet)
		if err != nil {
			return recoverySetID{}, nil, err
		}
		err = writeNextPacket(buf, setID, recoveryPacketType, padPacketBytes(recoveryPacketBytes))
		if err != nil {
			return recoverySetID{}, nil, err
		}
	}

	// TODO: Sort packet types before iterating.
	for packetType, packetBytesList := range file.unknownPackets {
		for _, packetBytes := range packetBytesList {
			err = writeNextPacket(buf, setID, packetType, packetBytes)
			if err != nil {
				return recoverySetID{}, nil, err
			}
		}
	}

	return setID, buf.Bytes(), nil
}
