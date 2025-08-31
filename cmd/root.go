package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"void-cutter/internal/audio"
	"void-cutter/internal/config"
	"void-cutter/internal/loudness"
	"void-cutter/internal/silence"

	"github.com/spf13/cobra"
)

var cfg *config.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "void-cutter [OPTIONS] <input1.wav> <input2.wav> ...",
	Short: "Audio editing tool for podcast production",
	Long: `void-cutter is a command-line tool for automatic audio editing of podcast recordings.
It performs loudness normalization to Apple Podcast standards (-16 LUFS) and cuts common
silence periods across multiple audio tracks.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runVoidCutter,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cfg = config.DefaultConfig()

	// Add flags based on CLI specification
	rootCmd.Flags().StringVarP(&cfg.OutputSuffix, "output-suffix", "s", cfg.OutputSuffix,
		"Suffix to append to output file names")
	rootCmd.Flags().Float64VarP(&cfg.TargetLoudness, "target-loudness", "l", cfg.TargetLoudness,
		"Target loudness value in LUFS")
	rootCmd.Flags().Float64VarP(&cfg.SilenceThreshold, "silence-threshold", "t", cfg.SilenceThreshold,
		"Silence threshold in dBFS (-120 to 0)")
	rootCmd.Flags().IntVarP(&cfg.MinSilenceDuration, "min-silence-duration", "m", cfg.MinSilenceDuration,
		"Minimum silence duration in milliseconds")
	rootCmd.Flags().IntVarP(&cfg.KeepSilenceDuration, "keep-silence-duration", "k", cfg.KeepSilenceDuration,
		"Duration of silence to keep after cutting in milliseconds")

	// Add debug mode flag
	rootCmd.Flags().Bool("debug-info", false,
		"Show detailed debug information about audio files")

	// Add test mode flag
	rootCmd.Flags().Bool("test-copy", false,
		"Test mode: only copy input to output without processing")
}

func runVoidCutter(cmd *cobra.Command, args []string) error {
	// Set input files from command line arguments
	cfg.InputFiles = args

	// Get test mode flag
	testMode, _ := cmd.Flags().GetBool("test-copy")

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate input files exist and are WAV files
	for _, file := range cfg.InputFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("input file not found: %s", file)
		}

		if !strings.HasSuffix(strings.ToLower(file), ".wav") {
			return fmt.Errorf("input file must be a WAV file: %s", file)
		}
	}

	fmt.Printf("void-cutter started with %d input files\n", len(cfg.InputFiles))
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Target Loudness: %.1f LUFS\n", cfg.TargetLoudness)
	fmt.Printf("  Silence Threshold: %.1f dBFS\n", cfg.SilenceThreshold)
	fmt.Printf("  Min Silence Duration: %d ms\n", cfg.MinSilenceDuration)
	fmt.Printf("  Keep Silence Duration: %d ms\n", cfg.KeepSilenceDuration)
	fmt.Printf("  Output Suffix: %s\n", cfg.OutputSuffix)
	fmt.Println()

	// Load all audio files
	fmt.Println("Loading audio files...")
	var audioFiles []*audio.AudioData

	for i, file := range cfg.InputFiles {
		fmt.Printf("[%d/%d] Loading: %s", i+1, len(cfg.InputFiles), file)

		audioData, err := audio.LoadWAV(file)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", file, err)
		}

		audioFiles = append(audioFiles, audioData)
		fmt.Printf(" ✓ (%.2fs, %dHz, %dch)\n", audioData.Duration, audioData.SampleRate, audioData.Channels)
	}

	// Validate audio compatibility
	fmt.Println("\nValidating audio compatibility...")
	if err := audio.ValidateAudioFiles(audioFiles); err != nil {
		return fmt.Errorf("audio validation failed: %w", err)
	}
	fmt.Println("✓ All audio files are compatible")

	// Show debug information if requested
	debugInfo, _ := cmd.Flags().GetBool("debug-info")
	if debugInfo {
		fmt.Println("\n=== DEBUG INFORMATION ===")
		for _, audioData := range audioFiles {
			audioData.AnalyzeContent()
		}
	}

	// Test mode: skip processing and just copy files
	if testMode {
		fmt.Println("\n🧪 TEST MODE: Copying files without processing...")

		// Generate output files (no processing)
		fmt.Println("\nGenerating output files...")
		for i, audioData := range audioFiles {
			outputFile := generateOutputFilename(cfg.InputFiles[i], cfg.OutputSuffix)
			fmt.Printf("[%d/%d] Saving: %s", i+1, len(audioFiles), outputFile)

			err := audioData.SaveWAV(outputFile)
			if err != nil {
				return fmt.Errorf("failed to save %s: %w", outputFile, err)
			}

			fmt.Printf(" ✓ (%.2fs)\n", audioData.Duration)
		}

		fmt.Printf("\n✅ Test copy completed successfully!\n")
		return nil
	}

	// Normal processing mode
	// Loudness measurement and normalization
	fmt.Println("\nMeasuring loudness...")
	var loudnessResults []*loudness.LoudnessResult

	for i, audioData := range audioFiles {
		fmt.Printf("[%d/%d] Measuring: %s", i+1, len(audioFiles), audioData.Filename)

		result, err := loudness.MeasureLoudness(audioData)
		if err != nil {
			return fmt.Errorf("failed to measure loudness for %s: %w", audioData.Filename, err)
		}

		loudnessResults = append(loudnessResults, result)
		fmt.Printf(" ✓ (%.1f LUFS)\n", result.IntegratedLoudness)
	}

	// Apply loudness normalization
	fmt.Printf("\nApplying loudness normalization (target: %.1f LUFS)...\n", cfg.TargetLoudness)
	normResults, err := loudness.NormalizeMultipleAudio(audioFiles, cfg.TargetLoudness)
	if err != nil {
		return fmt.Errorf("failed to normalize audio: %w", err)
	}

	// Print normalization summary
	loudness.PrintNormalizationSummary(normResults)

	// Detect common silence regions
	fmt.Println("\nDetecting common silence regions...")
	silenceConfig := silence.SilenceDetectionConfig{
		ThresholdDBFS: cfg.SilenceThreshold,
		MinDurationMs: cfg.MinSilenceDuration,
		ChunkSizeMs:   10, // 10ms chunks for analysis
	}

	detectionResult, err := silence.DetectCommonSilence(audioFiles, silenceConfig)
	if err != nil {
		return fmt.Errorf("failed to detect silence: %w", err)
	}

	detectionResult.Print()

	// Cut silence regions if any were found
	if len(detectionResult.CommonSilenceRegions) > 0 {
		fmt.Printf("\nCutting silence regions (keeping %d ms)...\n", cfg.KeepSilenceDuration)

		cuttingResults, err := silence.CutSilenceInMultipleFiles(
			audioFiles,
			detectionResult.CommonSilenceRegions,
			cfg.KeepSilenceDuration)
		if err != nil {
			return fmt.Errorf("failed to cut silence: %w", err)
		}

		silence.PrintCuttingSummary(cuttingResults)
	} else {
		fmt.Println("\nNo silence regions to cut.")
	}

	// Generate output files
	fmt.Println("\nGenerating output files...")
	for i, audioData := range audioFiles {
		outputFile := generateOutputFilename(cfg.InputFiles[i], cfg.OutputSuffix)
		fmt.Printf("[%d/%d] Saving: %s", i+1, len(audioFiles), outputFile)

		err := audioData.SaveWAV(outputFile)
		if err != nil {
			return fmt.Errorf("failed to save %s: %w", outputFile, err)
		}

		fmt.Printf(" ✓ (%.2fs)\n", audioData.Duration)
	}

	fmt.Printf("\n✅ Processing completed successfully!\n")
	fmt.Printf("Generated %d output file(s) with suffix '%s'\n", len(audioFiles), cfg.OutputSuffix)
	return nil
}

func generateOutputFilename(inputFile, suffix string) string {
	dir := filepath.Dir(inputFile)
	basename := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	ext := filepath.Ext(inputFile)
	return filepath.Join(dir, basename+suffix+ext)
}
