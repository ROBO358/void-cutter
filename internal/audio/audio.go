package audio

import (
	"fmt"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// AudioData represents decoded PCM audio data
type AudioData struct {
	Samples    []int32 // PCM samples (interleaved for multi-channel)
	SampleRate int     // Sample rate in Hz
	Channels   int     // Number of channels
	BitDepth   int     // Bit depth
	Duration   float64 // Duration in seconds
	Filename   string  // Original filename
}

// LoadWAV loads a WAV file and returns AudioData
func LoadWAV(filename string) (*AudioData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV file: %s", filename)
	}

	// Read the format information
	format := decoder.Format()
	if format == nil {
		return nil, fmt.Errorf("failed to read format from %s", filename)
	}

	fmt.Printf(" (loading...)")

	// Create audio buffer to hold the data
	intBuf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to decode PCM data from %s: %w", filename, err)
	}

	if intBuf == nil || intBuf.Data == nil {
		return nil, fmt.Errorf("no PCM data found in %s", filename)
	}

	// Convert to our internal format (int32)
	samples := make([]int32, len(intBuf.Data))
	for i, sample := range intBuf.Data {
		samples[i] = int32(sample)
	}

	duration := float64(len(samples)) / float64(format.SampleRate) / float64(format.NumChannels)

	// Get bit depth from the buffer (default to 16 if not available)
	bitDepth := 16
	if intBuf.SourceBitDepth > 0 {
		bitDepth = intBuf.SourceBitDepth
	}

	return &AudioData{
		Samples:    samples,
		SampleRate: int(format.SampleRate),
		Channels:   int(format.NumChannels),
		BitDepth:   bitDepth,
		Duration:   duration,
		Filename:   filename,
	}, nil
}

// SaveWAV saves AudioData to a WAV file
func (ad *AudioData) SaveWAV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	// Create encoder with the audio format
	encoder := wav.NewEncoder(file, ad.SampleRate, ad.BitDepth, ad.Channels, 1)

	// Convert samples back to audio.IntBuffer format
	intBuf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: ad.Channels,
			SampleRate:  ad.SampleRate,
		},
		Data:           make([]int, len(ad.Samples)),
		SourceBitDepth: ad.BitDepth,
	}

	for i, sample := range ad.Samples {
		intBuf.Data[i] = int(sample)
	}

	if err := encoder.Write(intBuf); err != nil {
		return fmt.Errorf("failed to write audio data to %s: %w", filename, err)
	}

	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to close encoder for %s: %w", filename, err)
	}

	return nil
}

// GetSampleCount returns the total number of samples
func (ad *AudioData) GetSampleCount() int {
	return len(ad.Samples)
}

// GetFrameCount returns the number of frames (samples per channel)
func (ad *AudioData) GetFrameCount() int {
	if ad.Channels == 0 {
		return 0
	}
	return len(ad.Samples) / ad.Channels
}