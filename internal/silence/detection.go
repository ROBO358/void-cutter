package silence

import (
	"fmt"
	"math"

	"void-cutter/internal/audio"
)

// SilenceRegion represents a detected silence region
type SilenceRegion struct {
	StartFrame int     // Starting frame (sample index / channels)
	EndFrame   int     // Ending frame (exclusive)
	Duration   float64 // Duration in seconds
	StartTime  float64 // Start time in seconds
	EndTime    float64 // End time in seconds
}

// SilenceDetectionConfig holds parameters for silence detection
type SilenceDetectionConfig struct {
	ThresholdDBFS float64 // Silence threshold in dBFS
	MinDurationMs int     // Minimum silence duration in milliseconds
	ChunkSizeMs   int     // Analysis chunk size in milliseconds (default: 10ms)
}

// DetectionResult contains the results of silence detection
type DetectionResult struct {
	CommonSilenceRegions []SilenceRegion   // Regions silent in ALL tracks
	IndividualSilence    [][]SilenceRegion // Per-file silence regions
	TotalCommonSilence   float64           // Total duration of common silence
	AudioFiles           []*audio.AudioData
	Config               SilenceDetectionConfig
}

// DetectCommonSilence finds silence regions that are common across all audio files
func DetectCommonSilence(audioFiles []*audio.AudioData, config SilenceDetectionConfig) (*DetectionResult, error) {
	if len(audioFiles) == 0 {
		return nil, fmt.Errorf("no audio files provided")
	}

	// Validate all files have same format
	reference := audioFiles[0]
	for _, audio := range audioFiles[1:] {
		if audio.SampleRate != reference.SampleRate || audio.Channels != reference.Channels {
			return nil, fmt.Errorf("all audio files must have the same sample rate and channel count")
		}
	}

	// Calculate chunk size in frames
	chunkFrames := (config.ChunkSizeMs * reference.SampleRate) / 1000
	if chunkFrames == 0 {
		chunkFrames = 1
	}

	// Find the shortest audio duration to analyze
	minFrames := reference.GetFrameCount()
	for _, audio := range audioFiles[1:] {
		if audio.GetFrameCount() < minFrames {
			minFrames = audio.GetFrameCount()
		}
	}

	fmt.Printf("Analyzing %d frames in chunks of %d frames (%.1fms)\n",
		minFrames, chunkFrames, float64(config.ChunkSizeMs))

	// Detect silence in each chunk across all files
	var commonSilenceRegions []SilenceRegion
	var currentSilenceStart = -1

	for frameStart := 0; frameStart < minFrames; frameStart += chunkFrames {
		frameEnd := frameStart + chunkFrames
		if frameEnd > minFrames {
			frameEnd = minFrames
		}

		// Check if this chunk is silent in ALL files
		isCommonSilence := true
		for _, audioData := range audioFiles {
			if !isChunkSilent(audioData, frameStart, frameEnd, config.ThresholdDBFS) {
				isCommonSilence = false
				break
			}
		}

		// Track silence regions
		if isCommonSilence {
			if currentSilenceStart == -1 {
				currentSilenceStart = frameStart
			}
		} else {
			// End of silence region
			if currentSilenceStart != -1 {
				silenceRegion := createSilenceRegion(currentSilenceStart, frameStart, reference.SampleRate)

				// Check if silence duration meets minimum requirement
				if silenceRegion.Duration >= float64(config.MinDurationMs)/1000.0 {
					commonSilenceRegions = append(commonSilenceRegions, silenceRegion)
				}
				currentSilenceStart = -1
			}
		}
	}

	// Handle silence region that extends to the end
	if currentSilenceStart != -1 {
		silenceRegion := createSilenceRegion(currentSilenceStart, minFrames, reference.SampleRate)
		if silenceRegion.Duration >= float64(config.MinDurationMs)/1000.0 {
			commonSilenceRegions = append(commonSilenceRegions, silenceRegion)
		}
	}

	// Calculate total common silence duration
	totalCommonSilence := 0.0
	for _, region := range commonSilenceRegions {
		totalCommonSilence += region.Duration
	}

	return &DetectionResult{
		CommonSilenceRegions: commonSilenceRegions,
		IndividualSilence:    nil, // Not implemented in this version
		TotalCommonSilence:   totalCommonSilence,
		AudioFiles:           audioFiles,
		Config:               config,
	}, nil
}

// isChunkSilent checks if a chunk of audio is below the silence threshold
func isChunkSilent(audioData *audio.AudioData, startFrame, endFrame int, thresholdDBFS float64) bool {
	if startFrame >= endFrame || endFrame > audioData.GetFrameCount() {
		return true // Consider out-of-bounds as silent
	}

	channels := audioData.Channels
	startSample := startFrame * channels
	endSample := endFrame * channels

	if endSample > len(audioData.Samples) {
		endSample = len(audioData.Samples)
	}

	// Calculate RMS for this chunk
	var sumSquares float64
	sampleCount := endSample - startSample

	// Get appropriate normalization factor based on bit depth
	var normalizationFactor float64
	switch audioData.BitDepth {
	case 16:
		normalizationFactor = 32768.0 // 2^15
	case 24:
		normalizationFactor = 8388608.0 // 2^23
	case 32:
		normalizationFactor = 2147483648.0 // 2^31
	default:
		normalizationFactor = 32768.0 // Default to 16-bit
	}

	for i := startSample; i < endSample; i++ {
		// Normalize to [-1, 1] range using appropriate factor
		normalized := float64(audioData.Samples[i]) / normalizationFactor
		sumSquares += normalized * normalized
	}

	if sampleCount == 0 {
		return true
	}

	rms := math.Sqrt(sumSquares / float64(sampleCount))

	// Convert to dBFS
	if rms == 0 {
		return true // Perfect silence
	}

	rmsDBFS := 20 * math.Log10(rms)

	return rmsDBFS <= thresholdDBFS
}

// createSilenceRegion creates a SilenceRegion from frame indices
func createSilenceRegion(startFrame, endFrame, sampleRate int) SilenceRegion {
	startTime := float64(startFrame) / float64(sampleRate)
	endTime := float64(endFrame) / float64(sampleRate)
	duration := endTime - startTime

	return SilenceRegion{
		StartFrame: startFrame,
		EndFrame:   endFrame,
		Duration:   duration,
		StartTime:  startTime,
		EndTime:    endTime,
	}
}

// Print displays the detection results
func (dr *DetectionResult) Print() {
	fmt.Printf("\nSilence Detection Results:\n")
	fmt.Printf("Threshold: %.1f dBFS\n", dr.Config.ThresholdDBFS)
	fmt.Printf("Min Duration: %d ms\n", dr.Config.MinDurationMs)
	fmt.Printf("Total Files: %d\n", len(dr.AudioFiles))
	fmt.Printf("\nCommon Silence Regions: %d\n", len(dr.CommonSilenceRegions))

	if len(dr.CommonSilenceRegions) == 0 {
		fmt.Printf("No common silence regions found with current settings.\n")
		return
	}

	for i, region := range dr.CommonSilenceRegions {
		fmt.Printf("[%d] %.2fs - %.2fs (%.2fs duration)\n",
			i+1, region.StartTime, region.EndTime, region.Duration)
	}

	fmt.Printf("\nTotal common silence: %.2fs (%.1f%% of audio)\n",
		dr.TotalCommonSilence,
		(dr.TotalCommonSilence/dr.AudioFiles[0].Duration)*100)
}

// DefaultSilenceConfig returns default silence detection configuration
func DefaultSilenceConfig() SilenceDetectionConfig {
	return SilenceDetectionConfig{
		ThresholdDBFS: -50.0,
		MinDurationMs: 500,
		ChunkSizeMs:   10,
	}
}
