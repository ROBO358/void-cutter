package silence

import (
	"fmt"

	"void-cutter/internal/audio"
)

// CuttingResult contains the results of silence cutting
type CuttingResult struct {
	OriginalDuration float64         // Original audio duration
	NewDuration      float64         // Duration after cutting
	RemovedDuration  float64         // Total duration removed
	RegionsCut       []SilenceRegion // Regions that were cut
	KeepDurationMs   int             // Duration kept per silence region
	Filename         string
}

// CutSilenceRegions removes or shortens silence regions in audio data
func CutSilenceRegions(audioData *audio.AudioData, silenceRegions []SilenceRegion, keepDurationMs int) (*CuttingResult, error) {
	if audioData == nil {
		return nil, fmt.Errorf("audio data is nil")
	}

	originalDuration := audioData.Duration
	keepDurationSec := float64(keepDurationMs) / 1000.0

	// Work backwards through regions to maintain correct indices
	var regionsCut []SilenceRegion
	var totalRemovedDuration float64

	// Clone the audio data to work with
	modifiedSamples := make([]int32, len(audioData.Samples))
	copy(modifiedSamples, audioData.Samples)

	// Process regions from end to beginning to maintain indices
	for i := len(silenceRegions) - 1; i >= 0; i-- {
		region := silenceRegions[i]

		// Skip if region is too short
		if region.Duration <= keepDurationSec {
			continue
		}

		// Calculate how much to cut
		cutDuration := region.Duration - keepDurationSec
		cutFrames := int(cutDuration * float64(audioData.SampleRate))
		cutSamples := cutFrames * audioData.Channels

		// Calculate cut positions
		startSample := region.StartFrame * audioData.Channels
		endSample := region.EndFrame * audioData.Channels
		keepSamples := int(keepDurationSec*float64(audioData.SampleRate)) * audioData.Channels

		// Ensure we don't exceed array bounds
		if endSample > len(modifiedSamples) {
			endSample = len(modifiedSamples)
		}
		if startSample < 0 {
			startSample = 0
		}

		// Cut from the middle/end of the silence region, keeping some at the start
		cutStartSample := startSample + keepSamples
		if cutStartSample > endSample {
			cutStartSample = startSample
			keepSamples = endSample - startSample
		}

		// Remove the samples
		if cutStartSample < endSample && cutSamples > 0 {
			actualCutSamples := endSample - cutStartSample
			if actualCutSamples > cutSamples {
				actualCutSamples = cutSamples
			}

			// Create new slice without the cut samples
			newSamples := make([]int32, len(modifiedSamples)-actualCutSamples)
			copy(newSamples[:cutStartSample], modifiedSamples[:cutStartSample])
			copy(newSamples[cutStartSample:], modifiedSamples[cutStartSample+actualCutSamples:])
			modifiedSamples = newSamples

			// Track what was cut
			actualCutDuration := float64(actualCutSamples) / float64(audioData.SampleRate) / float64(audioData.Channels)
			totalRemovedDuration += actualCutDuration

			regionsCut = append([]SilenceRegion{region}, regionsCut...) // Prepend to maintain order
		}
	}

	// Update audio data with modified samples
	audioData.Samples = modifiedSamples
	newDuration := float64(len(modifiedSamples)) / float64(audioData.SampleRate) / float64(audioData.Channels)
	audioData.Duration = newDuration

	return &CuttingResult{
		OriginalDuration: originalDuration,
		NewDuration:      newDuration,
		RemovedDuration:  totalRemovedDuration,
		RegionsCut:       regionsCut,
		KeepDurationMs:   keepDurationMs,
		Filename:         audioData.Filename,
	}, nil
}

// CutSilenceInMultipleFiles cuts silence in multiple audio files using the same regions
func CutSilenceInMultipleFiles(audioFiles []*audio.AudioData, silenceRegions []SilenceRegion, keepDurationMs int) ([]*CuttingResult, error) {
	if len(audioFiles) == 0 {
		return nil, fmt.Errorf("no audio files provided")
	}

	results := make([]*CuttingResult, len(audioFiles))

	for i, audioData := range audioFiles {
		result, err := CutSilenceRegions(audioData, silenceRegions, keepDurationMs)
		if err != nil {
			return nil, fmt.Errorf("failed to cut silence in %s: %w", audioData.Filename, err)
		}
		results[i] = result
	}

	return results, nil
}

// Print displays cutting results
func (cr *CuttingResult) Print() {
	fmt.Printf("Silence Cutting: %s\n", cr.Filename)
	fmt.Printf("  Original Duration: %.2fs\n", cr.OriginalDuration)
	fmt.Printf("  New Duration: %.2fs\n", cr.NewDuration)
	fmt.Printf("  Removed: %.2fs (%.1f%%)\n",
		cr.RemovedDuration,
		(cr.RemovedDuration/cr.OriginalDuration)*100)
	fmt.Printf("  Regions Cut: %d\n", len(cr.RegionsCut))
}

// PrintCuttingSummary displays a summary of all cutting results
func PrintCuttingSummary(results []*CuttingResult) {
	fmt.Printf("\nSilence Cutting Summary:\n")

	totalOriginal := 0.0
	totalNew := 0.0
	totalRemoved := 0.0

	for i, result := range results {
		fmt.Printf("[%d] %s: %.2fs → %.2fs (%.2fs removed)\n",
			i+1, result.Filename, result.OriginalDuration, result.NewDuration, result.RemovedDuration)

		totalOriginal += result.OriginalDuration
		totalNew += result.NewDuration
		totalRemoved += result.RemovedDuration
	}

	fmt.Printf("\nTotal Summary:\n")
	fmt.Printf("  Original Total: %.2fs\n", totalOriginal)
	fmt.Printf("  New Total: %.2fs\n", totalNew)
	fmt.Printf("  Total Removed: %.2fs (%.1f%%)\n",
		totalRemoved, (totalRemoved/totalOriginal)*100)
}
