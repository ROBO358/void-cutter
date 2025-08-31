package loudness

import (
	"fmt"
	"math"

	"void-cutter/internal/audio"
)

// LoudnessResult contains loudness measurement results
type LoudnessResult struct {
	IntegratedLoudness float64 // LUFS (Loudness Units relative to Full Scale)
	LoudnessRange      float64 // LU (Loudness Units)
	TruePeak           float64 // dBFS
	RMSLevel           float64 // dBFS (Root Mean Square level)
	Filename           string
}

// MeasureLoudness calculates loudness metrics for audio data
// This is a simplified implementation that approximates LUFS using RMS
// For production use, consider integrating with go-ebur128 for ITU-R BS.1770-4 compliance
func MeasureLoudness(audioData *audio.AudioData) (*LoudnessResult, error) {
	if audioData == nil {
		return nil, fmt.Errorf("audio data is nil")
	}

	if len(audioData.Samples) == 0 {
		return nil, fmt.Errorf("no audio samples found")
	}

	// Calculate RMS (Root Mean Square) with bit depth consideration
	rms := calculateRMSWithBitDepth(audioData.Samples, audioData.BitDepth)

	// Convert RMS to dBFS
	rmsDB := 20 * math.Log10(rms)

	// Approximate LUFS using RMS
	// This is a simplified conversion - actual LUFS requires complex filtering
	// According to ITU-R BS.1770-4, but this gives a reasonable approximation
	lufs := rmsDB - 0.691 // Rough calibration offset for LUFS

	// Calculate true peak with bit depth consideration
	truePeak := calculateTruePeakWithBitDepth(audioData.Samples, audioData.BitDepth)
	truePeakDB := 20 * math.Log10(truePeak)

	return &LoudnessResult{
		IntegratedLoudness: lufs,
		LoudnessRange:      0.0, // Not implemented in this simplified version
		TruePeak:           truePeakDB,
		RMSLevel:           rmsDB,
		Filename:           audioData.Filename,
	}, nil
}

// calculateRMSWithBitDepth computes RMS with proper bit depth normalization
func calculateRMSWithBitDepth(samples []int32, bitDepth int) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	// Calculate the proper maximum value based on bit depth
	var maxValue float64
	switch bitDepth {
	case 16:
		maxValue = 32768.0 // 2^15
	case 24:
		maxValue = 8388608.0 // 2^23
	case 32:
		maxValue = 2147483648.0 // 2^31
	default:
		maxValue = 32768.0 // Default to 16-bit
	}

	var sumSquares float64
	for _, sample := range samples {
		// Normalize to [-1, 1] range based on actual bit depth
		normalized := float64(sample) / maxValue
		// Clamp to valid range
		if normalized > 1.0 {
			normalized = 1.0
		} else if normalized < -1.0 {
			normalized = -1.0
		}
		sumSquares += normalized * normalized
	}

	return math.Sqrt(sumSquares / float64(len(samples)))
}

// calculateTruePeakWithBitDepth finds maximum absolute sample with proper bit depth normalization
func calculateTruePeakWithBitDepth(samples []int32, bitDepth int) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	// Calculate the proper maximum value based on bit depth
	var maxValue float64
	switch bitDepth {
	case 16:
		maxValue = 32767.0 // 2^15 - 1
	case 24:
		maxValue = 8388607.0 // 2^23 - 1
	case 32:
		maxValue = 2147483647.0 // 2^31 - 1
	default:
		maxValue = 32767.0 // Default to 16-bit
	}

	var maxAbsSample int32
	for _, sample := range samples {
		absSample := sample
		if absSample < 0 {
			absSample = -absSample
		}
		if absSample > maxAbsSample {
			maxAbsSample = absSample
		}
	}

	// Normalize to [0, 1] range based on actual bit depth
	peak := float64(maxAbsSample) / maxValue
	if peak > 1.0 {
		peak = 1.0
	}
	return peak
}

// CalculateGain computes the gain needed to reach target loudness
func CalculateGain(currentLUFS, targetLUFS float64) float64 {
	// Gain = 10^((TargetLUFS - MeasuredLUFS) / 20)
	return math.Pow(10, (targetLUFS-currentLUFS)/20.0)
}

// PrintLoudnessResult displays loudness measurement results
func (lr *LoudnessResult) Print() {
	fmt.Printf("Loudness Analysis: %s\n", lr.Filename)
	fmt.Printf("  Integrated Loudness: %.1f LUFS\n", lr.IntegratedLoudness)
	fmt.Printf("  RMS Level: %.1f dBFS\n", lr.RMSLevel)
	fmt.Printf("  True Peak: %.1f dBFS\n", lr.TruePeak)
	if lr.TruePeak > -0.1 {
		fmt.Printf("  ⚠️  Warning: True peak is close to 0dBFS (risk of clipping)\n")
	}
}

// ValidateTargetLoudness checks if the target loudness is reasonable
func ValidateTargetLoudness(targetLUFS float64) error {
	// Common loudness standards:
	// Apple Podcasts: -16 LUFS
	// Spotify: -14 LUFS
	// YouTube: -14 LUFS
	// EBU R128: -23 LUFS (broadcast)

	if targetLUFS > -6.0 {
		return fmt.Errorf("target loudness %.1f LUFS is too high (risk of severe clipping)", targetLUFS)
	}

	if targetLUFS < -30.0 {
		return fmt.Errorf("target loudness %.1f LUFS is too low (audio will be very quiet)", targetLUFS)
	}

	return nil
}
