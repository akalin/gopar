package par2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
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
	sliceByteCount int
	recoverySet    []fileID
	nonRecoverySet []fileID
}

func checkFileIDSetsSorted(recoverySet, nonRecoverySet []fileID) error {
	if !sort.SliceIsSorted(recoverySet, func(i, j int) bool {
		return fileIDLess(recoverySet[i], recoverySet[j])
	}) {
		return errors.New("recovery set IDs not sorted")
	}

	if !sort.SliceIsSorted(nonRecoverySet, func(i, j int) bool {
		return fileIDLess(nonRecoverySet[i], nonRecoverySet[j])
	}) {
		return errors.New("non-recovery set IDs not sorted")
	}

	return nil
}

func readMainPacket(body []byte) (mainPacket, error) {
	buf := bytes.NewBuffer(body)

	var h mainPacketHeader
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return mainPacket{}, err
	}

	maxInt := uint64(^uint(0) >> 1)
	if h.SliceSize == 0 || h.SliceSize%4 != 0 || h.SliceSize > maxInt {
		return mainPacket{}, errors.New("invalid slice size")
	}

	sliceByteCount := int(h.SliceSize)

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

	err = checkFileIDSetsSorted(recoverySet, nonRecoverySet)
	if err != nil {
		return mainPacket{}, err
	}

	return mainPacket{sliceByteCount, recoverySet, nonRecoverySet}, nil
}

func writeMainPacket(packet mainPacket) ([]byte, error) {
	if packet.sliceByteCount == 0 || packet.sliceByteCount%4 != 0 {
		return nil, errors.New("invalid slice byte count")
	}

	if len(packet.recoverySet) == 0 {
		return nil, errors.New("empty recovery set")
	}

	if int64(len(packet.recoverySet)) > int64(math.MaxUint32) {
		return nil, errors.New("recovery set too big")
	}

	err := checkFileIDSetsSorted(packet.recoverySet, packet.nonRecoverySet)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)

	h := mainPacketHeader{
		SliceSize:        uint64(packet.sliceByteCount),
		RecoverySetCount: uint32(len(packet.recoverySet)),
	}

	err = binary.Write(buf, binary.LittleEndian, h)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, packet.recoverySet)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, packet.nonRecoverySet)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
