package par2

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/akalin/gopar/rsec16"
	"github.com/stretchr/testify/require"
)

type testEncoderDelegate struct {
	tb testing.TB
}

func (d testEncoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {
	if d.tb != nil {
		d.tb.Logf("OnDataFileLoad(%d, %d, byteCount=%d, %s, %v)", i, n, byteCount, path, err)
	}
}

func (d testEncoderDelegate) OnIndexFileWrite(path string, byteCount int, err error) {
	if d.tb != nil {
		d.tb.Logf("OnIndexFileWrite(%s, %d, %v)", path, byteCount, err)
	}
}

func (d testEncoderDelegate) OnRecoveryFileWrite(start, count, total int, path string, dataByteCount, byteCount int, err error) {
	if d.tb != nil {
		d.tb.Logf("OnRecoveryFileWrite(start=%d, count=%d, total=%d, %s, dataByteCount=%d, byteCount=%d, %v)", start, count, total, path, dataByteCount, byteCount, err)
	}
}

func newEncoderForTest(t *testing.T, io testFileIO, paths []string, sliceByteCount, parityShardCount int) (*Encoder, error) {
	return newEncoder(io, testEncoderDelegate{t}, paths, sliceByteCount, parityShardCount, rsec16.DefaultNumGoroutines())
}

func TestEncodeParity(t *testing.T) {
	io := testFileIO{
		tb: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3},
			"file.r01": {0x5, 0x6, 0x7, 0x8},
			"file.r02": {0x9, 0xa, 0xb, 0xc},
			"file.r03": {0xd, 0xe},
			"file.r04": {0xf},
		},
	}

	paths := []string{"file.rar", "file.r01", "file.r02", "file.r03", "file.r04"}

	sliceByteCount := 4
	parityShardCount := 3
	encoder, err := newEncoderForTest(t, io, paths, sliceByteCount, parityShardCount)
	require.NoError(t, err)

	err = encoder.LoadFileData()
	require.NoError(t, err)

	err = encoder.ComputeParityData()
	require.NoError(t, err)

	var recoverySet []fileID
	dataShardsByID := make(map[fileID][][]byte)
	for filename, data := range io.fileData {
		fileID, _, _, fileDataShards := computeDataFileInfo(sliceByteCount, filename, data)
		recoverySet = append(recoverySet, fileID)
		dataShardsByID[fileID] = fileDataShards
	}

	sort.Slice(recoverySet, func(i, j int) bool {
		return fileIDLess(recoverySet[i], recoverySet[j])
	})

	var dataShards [][]byte
	for _, fileID := range recoverySet {
		dataShards = append(dataShards, dataShardsByID[fileID]...)
	}

	coder, err := rsec16.NewCoderPAR2Vandermonde(len(dataShards), parityShardCount, rsec16.DefaultNumGoroutines())
	require.NoError(t, err)

	computedParityShards := coder.GenerateParity(dataShards)
	require.Equal(t, computedParityShards, encoder.parityShards)
}

func TestWriteParity(t *testing.T) {
	io := testFileIO{
		tb: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3},
			"file.r01": {0x5, 0x6, 0x7, 0x8},
			"file.r02": {0x9, 0xa, 0xb, 0xc},
			"file.r03": {0xd, 0xe},
			"file.r04": {0xf},
		},
	}

	paths := []string{"file.rar", "file.r01", "file.r02", "file.r03", "file.r04"}

	sliceByteCount := 4
	parityShardCount := 100
	encoder, err := newEncoderForTest(t, io, paths, sliceByteCount, parityShardCount)
	require.NoError(t, err)

	err = encoder.LoadFileData()
	require.NoError(t, err)

	err = encoder.ComputeParityData()
	require.NoError(t, err)

	err = encoder.Write("parity.par2")
	require.NoError(t, err)

	decoder, err := newDecoderForTest(t, io, "parity.par2")
	require.NoError(t, err)

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	ok, err := decoder.Verify(true)
	require.NoError(t, err)
	require.True(t, ok)
}

func benchmarkWriteParity(b *testing.B, io testFileIO, paths []string, parityShardCount, sliceByteCount int) {
	for i := 0; i < b.N; i++ {
		encoder, err := newEncoder(io, testEncoderDelegate{nil}, paths, sliceByteCount, parityShardCount)
		require.NoError(b, err)

		err = encoder.LoadFileData()
		require.NoError(b, err)

		err = encoder.ComputeParityData()
		require.NoError(b, err)

		err = encoder.Write("parity.par2")
		require.NoError(b, err)

		b.SetBytes(int64(len(io.fileData["parity.par2"])))
	}
}

func sizeString(size int) string {
	if size%(1024*1024*1024) == 0 {
		return fmt.Sprintf("%dG", size/(1024*1024*1024))
	} else if size%(1024*1024) == 0 {
		return fmt.Sprintf("%dM", size/(1024*1024))
	} else if size%1024 == 0 {
		return fmt.Sprintf("%dK", size/1024)
	}
	return fmt.Sprintf("%d", size)
}

func BenchmarkWriteParity(b *testing.B) {
	rand := rand.New(rand.NewSource(1))

	totalByteCounts := []int{1024, 100 * 1024, 1024 * 1024, 10 * 1024 * 1024}
	sliceByteCounts := []int{4, 16, 128, 1024, 4 * 1024}
	parityShardCounts := []int{10, 100, 1000}
	fileCount := 16
	for _, totalByteCount := range totalByteCounts {
		io := testFileIO{
			tb:       nil,
			fileData: make(map[string][]byte),
		}
		var paths []string
		for i := 0; i < fileCount; i++ {
			path := fmt.Sprintf("file.%03d", i)
			io.fileData[path] = make([]byte, totalByteCount/fileCount)
			n, err := rand.Read(io.fileData[path])
			require.NoError(b, err)
			require.Equal(b, totalByteCount/fileCount, n)
			paths = append(paths, path)
		}

		for _, sliceByteCount := range sliceByteCounts {
			if sliceByteCount > totalByteCount {
				continue
			}

			dataShardCount := totalByteCount / sliceByteCount
			if dataShardCount > 30000 {
				continue
			}

			for _, parityShardCount := range parityShardCounts {
				if parityShardCount > 2*dataShardCount {
					continue
				}

				name := fmt.Sprintf("tb=%s,sb=%s,ps=%d", sizeString(totalByteCount), sizeString(sliceByteCount), parityShardCount)
				b.Run(name, func(b *testing.B) {
					benchmarkWriteParity(b, io, paths, parityShardCount, sliceByteCount)
				})
			}
		}
	}
}
