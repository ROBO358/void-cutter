package audio

import (
	"fmt"
	"math"
)

// ApplyGain applies a gain factor to the audio samples
func (ad *AudioData) ApplyGain(gain float64) {
	// Calculate the maximum value based on bit depth
	var maxValue int32
	switch ad.BitDepth {
	case 16:
		maxValue = 32767 // 2^15 - 1
	case 24:
		maxValue = 8388607 // 2^23 - 1
	case 32:
		maxValue = 2147483647 // 2^31 - 1
	default:
		maxValue = 32767 // Default to 16-bit
	}

	minValue := -maxValue - 1

	clippedSamples := 0
	for i := range ad.Samples {
		newSample := float64(ad.Samples[i]) * gain

		// Clamp to prevent overflow based on actual bit depth
		if newSample > float64(maxValue) {
			ad.Samples[i] = maxValue
			clippedSamples++
		} else if newSample < float64(minValue) {
			ad.Samples[i] = minValue
			clippedSamples++
		} else {
			ad.Samples[i] = int32(newSample)
		}
	}

	// Warn if clipping occurred
	if clippedSamples > 0 {
		clippingPercentage := float64(clippedSamples) / float64(len(ad.Samples)) * 100
		fmt.Printf("  ⚠️  Clipped %d samples (%.2f%%) in %s\n",
			clippedSamples, clippingPercentage, ad.Filename)
	}
}

// Clone creates a deep copy of AudioData
func (ad *AudioData) Clone() *AudioData {
	samples := make([]int32, len(ad.Samples))
	copy(samples, ad.Samples)

	return &AudioData{
		Samples:    samples,
		SampleRate: ad.SampleRate,
		Channels:   ad.Channels,
		BitDepth:   ad.BitDepth,
		Duration:   ad.Duration,
		Filename:   ad.Filename,
	}
}

// AnalyzeContent provides detailed analysis of audio content
func (ad *AudioData) AnalyzeContent() {
	fmt.Printf("\n=== Audio Content Analysis: %s ===\n", ad.Filename)
	fmt.Printf("Basic Info:\n")
	fmt.Printf("  Duration: %.2f seconds\n", ad.Duration)
	fmt.Printf("  Sample Rate: %d Hz\n", ad.SampleRate)
	fmt.Printf("  Channels: %d\n", ad.Channels)
	fmt.Printf("  Bit Depth: %d bits\n", ad.BitDepth)
	fmt.Printf("  Total Samples: %d\n", len(ad.Samples))
	fmt.Printf("  Frames: %d\n", ad.GetFrameCount())

	if len(ad.Samples) == 0 {
		fmt.Printf("  ⚠️  No audio samples found!\n")
		return
	}

	// Sample value analysis
	var minSample, maxSample int32 = ad.Samples[0], ad.Samples[0]
	var zeroSamples, nonZeroSamples int
	var sumSquares float64

	for _, sample := range ad.Samples {
		if sample < minSample {
			minSample = sample
		}
		if sample > maxSample {
			maxSample = sample
		}
		if sample == 0 {
			zeroSamples++
		} else {
			nonZeroSamples++
		}

		// Calculate for RMS
		normalized := float64(sample) / 2147483648.0
		sumSquares += normalized * normalized
	}

	rms := math.Sqrt(sumSquares / float64(len(ad.Samples)))
	rmsDB := -math.Inf(1)
	if rms > 0 {
		rmsDB = 20 * math.Log10(rms)
	}

	fmt.Printf("\nSample Analysis:\n")
	fmt.Printf("  Min Sample: %d\n", minSample)
	fmt.Printf("  Max Sample: %d\n", maxSample)
	fmt.Printf("  Zero Samples: %d (%.1f%%)\n", zeroSamples, float64(zeroSamples)/float64(len(ad.Samples))*100)
	fmt.Printf("  Non-Zero Samples: %d (%.1f%%)\n", nonZeroSamples, float64(nonZeroSamples)/float64(len(ad.Samples))*100)
	fmt.Printf("  RMS Level: %.1f dBFS\n", rmsDB)

	// Check if file is mostly silent
	silencePercent := float64(zeroSamples) / float64(len(ad.Samples)) * 100
	if silencePercent > 95 {
		fmt.Printf("  🔇 WARNING: File is %.1f%% silent - may be empty or very quiet recording\n", silencePercent)
	} else if silencePercent > 80 {
		fmt.Printf("  ⚠️  File has %.1f%% silence - possibly a quiet recording\n", silencePercent)
	} else {
		fmt.Printf("  ✓ File contains %.1f%% audio content\n", 100-silencePercent)
	}

	// Analyze first and last seconds
	sampleRate := ad.SampleRate * ad.Channels
	if len(ad.Samples) >= sampleRate {
		fmt.Printf("\nContent Distribution:\n")

		// First second
		firstSecondNonZero := 0
		for i := 0; i < sampleRate && i < len(ad.Samples); i++ {
			if ad.Samples[i] != 0 {
				firstSecondNonZero++
			}
		}

		// Last second
		lastSecondNonZero := 0
		start := len(ad.Samples) - sampleRate
		if start < 0 {
			start = 0
		}
		for i := start; i < len(ad.Samples); i++ {
			if ad.Samples[i] != 0 {
				lastSecondNonZero++
			}
		}

		fmt.Printf("  First second: %.1f%% audio\n", float64(firstSecondNonZero)/float64(sampleRate)*100)
		fmt.Printf("  Last second: %.1f%% audio\n", float64(lastSecondNonZero)/float64(sampleRate)*100)
	}

	fmt.Printf("=====================================\n")
}
