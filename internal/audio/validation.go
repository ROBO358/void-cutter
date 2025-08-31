package audio

import (
	"fmt"
)

// ValidateAudioFiles checks if all audio files have compatible formats
func ValidateAudioFiles(audioFiles []*AudioData) error {
	if len(audioFiles) == 0 {
		return fmt.Errorf("no audio files provided")
	}

	reference := audioFiles[0]

	for i, audio := range audioFiles[1:] {
		if audio.SampleRate != reference.SampleRate {
			return fmt.Errorf("sample rate mismatch: file %s (%d Hz) vs %s (%d Hz)",
				audio.Filename, audio.SampleRate, reference.Filename, reference.SampleRate)
		}

		if audio.Channels != reference.Channels {
			return fmt.Errorf("channel count mismatch: file %s (%d channels) vs %s (%d channels)",
				audio.Filename, audio.Channels, reference.Filename, reference.Channels)
		}

		// Allow some tolerance in duration (within 1 second)
		durationDiff := audio.Duration - reference.Duration
		if durationDiff < 0 {
			durationDiff = -durationDiff
		}
		if durationDiff > 1.0 {
			return fmt.Errorf("significant duration difference: file %s (%.2fs) vs %s (%.2fs)",
				audio.Filename, audio.Duration, reference.Filename, reference.Duration)
		}

		fmt.Printf("Audio file %d validated: %s (%.2fs, %dHz, %dch)\n",
			i+2, audio.Filename, audio.Duration, audio.SampleRate, audio.Channels)
	}

	fmt.Printf("Reference audio: %s (%.2fs, %dHz, %dch)\n",
		reference.Filename, reference.Duration, reference.SampleRate, reference.Channels)

	return nil
}

// PrintAudioInfo displays information about an audio file
func (ad *AudioData) PrintInfo() {
	fmt.Printf("Audio File: %s\n", ad.Filename)
	fmt.Printf("  Duration: %.2f seconds\n", ad.Duration)
	fmt.Printf("  Sample Rate: %d Hz\n", ad.SampleRate)
	fmt.Printf("  Channels: %d\n", ad.Channels)
	fmt.Printf("  Bit Depth: %d bits\n", ad.BitDepth)
	fmt.Printf("  Total Samples: %d\n", len(ad.Samples))
	fmt.Printf("  Frames: %d\n", ad.GetFrameCount())
}
