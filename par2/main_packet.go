package par2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"sort"
)

var mainPacketType = packetType{'P', 'A', 'R', ' ', '2', '.', '0', '\x00', 'M', 'a', 'i', 'n'}

type mainPacketHeader struct {
	SliceSize        uint64
	RecoverySetCount uint32
}

type fileID [16]byte

func fileIDLess(id1, id2 fileID) bool {
	for i := len(id1) - 1; i >= 0; i-- {
		if id1[i] < id2[i] {
			return true
		} else if id1[i] > id2[i] {
			return false
		}
	}

	return false
}

type mainPacket struct {
	sliceByteCount uint64
	recoverySet    []fileID
	nonRecoverySet []fileID
}

func readMainPacket(body []byte) (mainPacket, error) {
	buf := bytes.NewBuffer(body)

	var h mainPacketHeader
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return mainPacket{}, err
	}

	if h.SliceSize == 0 || h.SliceSize%4 != 0 {
		return mainPacket{}, errors.New("invalid slice size")
	}

	if h.RecoverySetCount == 0 {
		return mainPacket{}, errors.New("empty recovery set")
	}

	fileIDSize := int(reflect.TypeOf(fileID{}).Size())
	if buf.Len()%fileIDSize != 0 {
		return mainPacket{}, errors.New("invalid size")
	}
	fileIDs := make([]fileID, buf.Len()/fileIDSize)
	err = binary.Read(buf, binary.LittleEndian, fileIDs)
	if err != nil {
		return mainPacket{}, err
	}

	if uint64(len(fileIDs)) < uint64(h.RecoverySetCount) {
		return mainPacket{}, errors.New("not enough file IDs")
	}

	recoverySet := fileIDs[:int(h.RecoverySetCount)]
	nonRecoverySet := fileIDs[int(h.RecoverySetCount):]

	if !sort.SliceIsSorted(recoverySet, func(i, j int) bool {
		return fileIDLess(recoverySet[i], recoverySet[j])
	}) {
		return mainPacket{}, errors.New("recovery set IDs not sorted")
	}

	if !sort.SliceIsSorted(nonRecoverySet, func(i, j int) bool {
		return fileIDLess(nonRecoverySet[i], nonRecoverySet[j])
	}) {
		return mainPacket{}, errors.New("non-recovery set IDs not sorted")
	}

	return mainPacket{h.SliceSize, recoverySet, nonRecoverySet}, nil
}
