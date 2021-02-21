package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func rms(val []float64) float64 {
	var sum float64
	for _, s := range val {
		sum += s
	}
	return sum / float64(len(val))
}

// WAV is a struct to hold wave file format data
type WAV struct {
	chunkID       [4]byte
	chunkSize     uint32
	format        [4]byte
	subchunk1ID   [4]byte
	subchunk1Size uint32
	audioFormat   uint16
	numChannels   uint16
	sampleRate    uint32
	byteRate      uint32
	blockAlign    uint16
	bitsPerSample uint16
	subchunk2ID   [4]byte
	subchunk2Size uint32
	data          [][]byte
	// Calculated fields
	NumSamples uint32
	SampleSize uint16
	Duration   float64
}

// NewWAV creates new WAV object and returns a pointer to it
func NewWAV() *WAV {
	return &WAV{}
}

func (object *WAV) read(r io.Reader) {
	binary.Read(r, binary.BigEndian, &object.chunkID)
	binary.Read(r, binary.LittleEndian, &object.chunkSize)
	binary.Read(r, binary.BigEndian, &object.format)
	binary.Read(r, binary.BigEndian, &object.subchunk1ID)
	binary.Read(r, binary.LittleEndian, &object.subchunk1Size)
	binary.Read(r, binary.LittleEndian, &object.audioFormat)
	binary.Read(r, binary.LittleEndian, &object.numChannels)
	binary.Read(r, binary.LittleEndian, &object.sampleRate)
	binary.Read(r, binary.LittleEndian, &object.byteRate)
	binary.Read(r, binary.LittleEndian, &object.blockAlign)
	binary.Read(r, binary.LittleEndian, &object.bitsPerSample)
	binary.Read(r, binary.BigEndian, &object.subchunk2ID)
	binary.Read(r, binary.LittleEndian, &object.subchunk2Size)
	object.NumSamples = (8 * object.subchunk2Size) / uint32((object.numChannels * object.bitsPerSample))
	object.SampleSize = (object.numChannels * object.bitsPerSample) / 8
	object.Duration = float64(object.subchunk2Size) / float64(object.byteRate)
	for i := 0; i < int(object.NumSamples); i++ {
		x := make([]byte, int(object.SampleSize))
		binary.Read(r, binary.LittleEndian, &x)
		object.data = append(object.data, x)
	}
}

func (object *WAV) readFile(path string) {
	f, err := os.Open(path)
	check(err)
	defer f.Close()
	object.read(f)
}

func (object *WAV) write(r io.Writer) {
	binary.Write(r, binary.BigEndian, object.chunkID)
	binary.Write(r, binary.LittleEndian, object.chunkSize)
	binary.Write(r, binary.BigEndian, object.format)
	binary.Write(r, binary.BigEndian, object.subchunk1ID)
	binary.Write(r, binary.LittleEndian, object.subchunk1Size)
	binary.Write(r, binary.LittleEndian, object.audioFormat)
	binary.Write(r, binary.LittleEndian, object.numChannels)
	binary.Write(r, binary.LittleEndian, object.sampleRate)
	binary.Write(r, binary.LittleEndian, object.byteRate)
	binary.Write(r, binary.LittleEndian, object.blockAlign)
	binary.Write(r, binary.LittleEndian, object.bitsPerSample)
	binary.Write(r, binary.BigEndian, object.subchunk2ID)
	binary.Write(r, binary.LittleEndian, object.subchunk2Size)
	for i := 0; i < len(object.data); i++ {
		binary.Write(r, binary.LittleEndian, object.data[i])
	}
}

func (object *WAV) writeFile(path string) {
	f, err := os.Create(path)
	check(err)
	defer f.Close()
	object.write(f)
}

func (object *WAV) dumpHeader(more bool) {
	fmt.Printf("File size: %.2fKB\n", float64(object.chunkSize)/1000)
	fmt.Printf("Number of samples: %d\n", object.NumSamples)
	fmt.Printf("Size of each sample: %d bytes\n", object.SampleSize)
	fmt.Printf("Duration of file: %fs\n", object.Duration)
	if more {
		fmt.Printf("%-14s %s\n", "chunkID:", object.chunkID)
		fmt.Printf("%-14s %d\n", "chunkSize:", object.chunkSize)
		fmt.Printf("%-14s %s\n", "format:", object.format)
		fmt.Printf("%-14s %s\n", "subchunk1ID:", object.subchunk1ID)
		fmt.Printf("%-14s %d\n", "subchunk1Size:", object.subchunk1Size)
		fmt.Printf("%-14s %d\n", "audioFormat:", object.audioFormat)
		fmt.Printf("%-14s %d\n", "numChannels:", object.numChannels)
		fmt.Printf("%-14s %d\n", "sampleRate:", object.sampleRate)
		fmt.Printf("%-14s %d\n", "byteRate:", object.byteRate)
		fmt.Printf("%-14s %d\n", "blockAlign:", object.blockAlign)
		fmt.Printf("%-14s %d\n", "bitsPerSample:", object.bitsPerSample)
		fmt.Printf("%-14s %s\n", "subchunk2ID:", object.subchunk2ID)
		fmt.Printf("%-14s %d\n", "subchunk2Size:", object.subchunk2Size)
	}
}

func (object *WAV) mix(t1 *WAV, t2 *WAV) {
	var longerTrack, shorterTrack *WAV
	if t1.NumSamples >= t2.NumSamples {
		longerTrack = t1
		shorterTrack = t2
	} else {
		longerTrack = t2
		shorterTrack = t1
	}
	*object = *longerTrack
	for i := 0; i < int(longerTrack.NumSamples); i++ {
		signal := make([]byte, object.SampleSize)
		var x int32
		if i < int(shorterTrack.NumSamples) {
			x = int32(int16(binary.LittleEndian.Uint16(longerTrack.data[i]))) + int32(int16(binary.LittleEndian.Uint16(shorterTrack.data[i])))
		} else {
			x = int32(int16(binary.LittleEndian.Uint16(longerTrack.data[i])))
		}
		if x > 32767 {
			x = 32767
		} else if x < -32768 {
			x = -32768
		}
		binary.LittleEndian.PutUint16(signal, uint16(x))
		object.data[i] = signal
	}
}

func (object *WAV) normalize(desiredPeak float64) {
	base := math.Pow(2, float64(object.bitsPerSample-1)) * math.Pow(10, (desiredPeak/20))
	var peak float64 = 0
	for i := 0; i < int(object.NumSamples); i++ {
		x := math.Abs(float64(int16(binary.LittleEndian.Uint16(object.data[i]))))
		if x > peak {
			peak = x
		}
	}
	normNum := base / peak
	for i := 0; i < int(object.NumSamples); i++ {
		signal := make([]byte, object.SampleSize)
		x := float64(int16(binary.LittleEndian.Uint16(object.data[i])))
		x *= normNum
		binary.LittleEndian.PutUint16(signal, uint16(x))
		object.data[i] = signal
	}
}

func (object *WAV) compress(thresh float64, R int, makeup float64, kneeWidth float64) {
	T := math.Pow(2, float64(object.bitsPerSample-1)) * math.Pow(10, (thresh/20))
	W := math.Pow(2, float64(object.bitsPerSample-1)) * math.Pow(10, (kneeWidth/20))
	// var period []float64
	for i := 0; i < int(object.NumSamples); i++ {
		signed := false
		signal := make([]byte, object.SampleSize)
		x := float64(int16(binary.LittleEndian.Uint16(object.data[i])))
		if x < 0 {
			signed = true
			x = math.Abs(x)
		}
		// ----- RMS CALC
		// if len(period) == int(object.sampleRate)/2000 {
		// 	period = period[1:]
		// }
		// period = append(period, x)
		// sigRMS := rms(period)

		// ----- HARD RMS
		// if sigRMS > T {
		// 	if x >= 0 {
		// 		x = T + (x-T)/float64(R)
		// 	} else if x < 0 {
		// 		x = -(T + (math.Abs(x)-T)/float64(R))
		// 	}
		// }

		// ----- HARD PEAK
		// if x > T {
		// 	x = T + (x-T)/float64(R)
		// } else if x < -T {
		// 	x = -(T + (math.Abs(x)-T)/float64(R))
		// }

		// ------ SMOOTH PEAK
		if x-T < -W/2 {
		} else if math.Abs(x-T) <= W/2 {
			x = x + ((1/float64(R)-1)*math.Pow(x-T+W/2, 2))/(W*2)
		} else if x-T > W/2 {
			x = T + (x-T)/float64(R)
		}

		if signed {
			x *= -1
		}
		binary.LittleEndian.PutUint16(signal, uint16(x))
		object.data[i] = signal
	}
	if makeup != 1.0 {
		fmt.Printf("Normalizing...\n")
		object.normalize(makeup)
	}
}
