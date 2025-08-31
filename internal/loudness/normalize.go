package loudness

import (
	"fmt"
	"math"

	"void-cutter/internal/audio"
)

// NormalizationResult contains the results of loudness normalization
type NormalizationResult struct {
	OriginalLoudness float64
	TargetLoudness   float64
	AppliedGain      float64
	GainDB           float64
	ClippingRisk     bool
	Filename         string
}

// NormalizeAudio applies loudness normalization to audio data
func NormalizeAudio(audioData *audio.AudioData, targetLUFS float64) (*NormalizationResult, error) {
	// Validate target loudness
	if err := ValidateTargetLoudness(targetLUFS); err != nil {
		return nil, fmt.Errorf("invalid target loudness: %w", err)
	}

	// Measure current loudness
	loudnessResult, err := MeasureLoudness(audioData)
	if err != nil {
		return nil, fmt.Errorf("failed to measure loudness: %w", err)
	}

	// Calculate required gain
	gain := CalculateGain(loudnessResult.IntegratedLoudness, targetLUFS)
	gainDB := 20 * math.Log10(gain)

	// Check for potential clipping
	clippingRisk := false
	if loudnessResult.TruePeak+gainDB > -0.1 {
		clippingRisk = true
		// Reduce gain to prevent severe clipping
		if gainDB > 6.0 { // If gain is more than 6dB, limit it
			fmt.Printf("  ⚠️  Limiting gain from %.1f dB to 6.0 dB to prevent severe clipping\n", gainDB)
			gainDB = 6.0
			gain = math.Pow(10, gainDB/20.0)
		}
	}

	// Apply gain to audio data
	audioData.ApplyGain(gain)

	result := &NormalizationResult{
		OriginalLoudness: loudnessResult.IntegratedLoudness,
		TargetLoudness:   targetLUFS,
		AppliedGain:      gain,
		GainDB:           gainDB,
		ClippingRisk:     clippingRisk,
		Filename:         audioData.Filename,
	}

	return result, nil
}

// NormalizeMultipleAudio normalizes multiple audio files to the same target loudness
func NormalizeMultipleAudio(audioFiles []*audio.AudioData, targetLUFS float64) ([]*NormalizationResult, error) {
	if len(audioFiles) == 0 {
		return nil, fmt.Errorf("no audio files provided")
	}

	results := make([]*NormalizationResult, len(audioFiles))

	for i, audioData := range audioFiles {
		result, err := NormalizeAudio(audioData, targetLUFS)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize %s: %w", audioData.Filename, err)
		}
		results[i] = result
	}

	return results, nil
}

// Print displays normalization results
func (nr *NormalizationResult) Print() {
	fmt.Printf("Normalization: %s\n", nr.Filename)
	fmt.Printf("  Original Loudness: %.1f LUFS\n", nr.OriginalLoudness)
	fmt.Printf("  Target Loudness: %.1f LUFS\n", nr.TargetLoudness)
	fmt.Printf("  Applied Gain: %.2f (%.1f dB)\n", nr.AppliedGain, nr.GainDB)

	if nr.ClippingRisk {
		fmt.Printf("  ⚠️  Warning: Potential clipping detected\n")
	} else {
		fmt.Printf("  ✓ No clipping risk\n")
	}
}

// PrintNormalizationSummary displays a summary of all normalization results
func PrintNormalizationSummary(results []*NormalizationResult) {
	fmt.Printf("\nLoudness Normalization Summary:\n")
	fmt.Printf("Target: %.1f LUFS\n", results[0].TargetLoudness)

	totalClippingRisk := 0
	for i, result := range results {
		fmt.Printf("[%d] %s: %.1f → %.1f LUFS (%.1f dB)",
			i+1, result.Filename, result.OriginalLoudness, result.TargetLoudness, result.GainDB)

		if result.ClippingRisk {
			fmt.Printf(" ⚠️")
			totalClippingRisk++
		} else {
			fmt.Printf(" ✓")
		}
		fmt.Println()
	}

	if totalClippingRisk > 0 {
		fmt.Printf("\n⚠️  %d file(s) have potential clipping risk\n", totalClippingRisk)
	} else {
		fmt.Printf("\n✓ All files normalized successfully without clipping risk\n")
	}
}
