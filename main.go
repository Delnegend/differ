package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "image/gif"
	_ "image/jpeg"
)

// loadImage opens and decodes an image file.
func loadImage(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image %s: %w", filePath, err)
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", filePath, err)
	}
	fmt.Printf("Loaded image %s (format: %s)\n", filePath, format)
	return img, nil
}

// generateOutputFilename creates the full output path based on the input path and a suffix.
// Example: generateOutputFilename("path/to/image.png", "BASE") -> "path/to/image.BASE.png"
// Example: generateOutputFilename("path/to/image.jpg", "DIFF") -> "path/to/image.DIFF.jpg"
func generateOutputFilename(inputPath, suffix string) (string, error) {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	baseName := filepath.Base(inputPath)
	if len(ext) > 0 {
		baseName = strings.TrimSuffix(baseName, ext)
	} else {
		// Handle case where there might not be an extension, though unlikely for images
		ext = ".png" // Default to png if no extension? Or maybe error? Let's default for now.
		fmt.Fprintf(os.Stderr, "Warning: Input file %s has no extension, assuming %s for output.\n", inputPath, ext)
	}
	// Ensure suffix starts with a dot if not empty
	if suffix != "" && !strings.HasPrefix(suffix, ".") {
		suffix = "." + suffix
	}

	outputBaseName := fmt.Sprintf("%s%s%s", baseName, suffix, ext)
	return filepath.Join(dir, outputBaseName), nil
}

// generateOriginalFilename reconstructs the full original filename path by removing suffixes.
// Example: generateOriginalFilename("path/to/image.BASE.png") -> "path/to/image.png"
// Example: generateOriginalFilename("path/to/image.DIFF.jpg") -> "path/to/image.jpg"
func generateOriginalFilename(inputPath string) (string, error) {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	baseName := filepath.Base(inputPath)
	nameWithoutExt := baseName
	if len(ext) > 0 {
		nameWithoutExt = strings.TrimSuffix(baseName, ext)
	} else {
		return "", fmt.Errorf("input file %s seems to be missing an extension", inputPath)
	}

	originalName := nameWithoutExt
	if strings.HasSuffix(originalName, ".BASE") {
		originalName = strings.TrimSuffix(originalName, ".BASE")
	} else if strings.HasSuffix(originalName, ".DIFF") {
		originalName = strings.TrimSuffix(originalName, ".DIFF")
	} else {
		// If no known suffix, maybe it's already the original? Or an error?
		// Let's assume it might be an error in usage, but proceed cautiously.
		fmt.Fprintf(os.Stderr, "Warning: Input file %s does not have expected .BASE or .DIFF suffix.\n", inputPath)
	}

	if originalName == "" {
		return "", fmt.Errorf("could not determine original base name for %s", inputPath)
	}

	originalFullName := fmt.Sprintf("%s%s", originalName, ext)
	return filepath.Join(dir, originalFullName), nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file from %s to %s: %w", src, dst, err)
	}
	return destFile.Sync() // Ensure data is written to stable storage
}

// createDiffImage compares two images and returns an RGBA image holding the differences.
func createDiffImage(baseImg, currentImg image.Image) (*image.RGBA, int) {
	bounds := baseImg.Bounds() // Assumes dimensions are already checked
	diffImg := image.NewRGBA(bounds)
	width, height := bounds.Dx(), bounds.Dy()
	diffPixels := 0

	for y := range height {
		for x := range width {
			absX, absY := bounds.Min.X+x, bounds.Min.Y+y
			c1 := color.RGBAModel.Convert(baseImg.At(absX, absY)).(color.RGBA)
			c2 := color.RGBAModel.Convert(currentImg.At(absX, absY)).(color.RGBA)

			if c1.R != c2.R || c1.G != c2.G || c1.B != c2.B {
				diffImg.Set(absX, absY, color.RGBA{R: c2.R, G: c2.G, B: c2.B, A: 255})
				diffPixels++
			}
		}
	}
	return diffImg, diffPixels
}

// processPair compares two images, creates a diff image, and saves it next to the current image.
// Designed to be run in a goroutine.
func processPair(wg *sync.WaitGroup, prevImg image.Image, currentImg image.Image, prevPath, currentPath string) {
	defer wg.Done() // Signal completion when this function returns

	fmt.Printf("Processing pair: %s vs %s\n", prevPath, currentPath)

	// Check dimensions
	bounds1 := prevImg.Bounds()
	bounds2 := currentImg.Bounds()
	if bounds1 != bounds2 {
		fmt.Fprintf(os.Stderr, "Error: Image dimensions do not match (%s vs %s) for pair (%s, %s). Skipping.\n",
			bounds1, bounds2, prevPath, currentPath)
		return
	}

	// Create difference image
	diffImg, diffPixels := createDiffImage(prevImg, currentImg)
	fmt.Printf("Found %d different pixels between %s and %s.\n", diffPixels, prevPath, currentPath)

	// Generate output filename for the *current* image's diff, placing it in the same directory
	diffOutputName, err := generateOutputFilename(currentPath, "DIFF")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating diff filename for %s: %v. Skipping save.\n", currentPath, err)
		return
	}

	// Save the difference image
	outFile, err := os.Create(diffOutputName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output file %s: %v\n", diffOutputName, err)
		return // Return on error
	}
	defer outFile.Close() // Ensure file is closed even on encode error

	err = png.Encode(outFile, diffImg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode difference image to PNG %s: %v\n", diffOutputName, err)
		// File will be closed by defer
	} else {
		fmt.Printf("Difference image saved to %s\n", diffOutputName)
	}
}

// applyDiff combines a base image and a diff image (where diff pixels overwrite base).
// Returns the newly reconstructed image.
func applyDiff(baseImg, diffImg image.Image) (*image.RGBA, error) {
	boundsBase := baseImg.Bounds()
	boundsDiff := diffImg.Bounds()

	if boundsBase != boundsDiff {
		return nil, fmt.Errorf("dimensions mismatch between base (%s) and diff (%s)", boundsBase, boundsDiff)
	}

	reconstructed := image.NewRGBA(boundsBase)
	width, height := boundsBase.Dx(), boundsBase.Dy()

	for y := range height {
		for x := range width {
			absX, absY := boundsBase.Min.X+x, boundsBase.Min.Y+y

			diffPixelColor := color.RGBAModel.Convert(diffImg.At(absX, absY)).(color.RGBA)

			// If the diff pixel has non-zero alpha, use its color. Otherwise, use the base image color.
			if diffPixelColor.A > 0 {
				reconstructed.Set(absX, absY, diffPixelColor)
			} else {
				basePixelColor := color.RGBAModel.Convert(baseImg.At(absX, absY))
				reconstructed.Set(absX, absY, basePixelColor)
			}
		}
	}
	return reconstructed, nil
}

// runDiffMode handles the logic for creating BASE and DIFF files concurrently.
func runDiffMode(inputFiles []string) {
	if len(inputFiles) < 2 {
		fmt.Fprintln(os.Stderr, "Error: -diff mode requires at least two input images.")
		printUsage()
		os.Exit(1)
	}

	// 1. Handle the first image (copy as BASE) - remains sequential
	firstImagePath := inputFiles[0]
	baseOutputName, err := generateOutputFilename(firstImagePath, "BASE") // Pass full path
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating base filename for %s: %v\n", firstImagePath, err)
		os.Exit(1)
	}
	err = copyFile(firstImagePath, baseOutputName) // Use full output path
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying base image %s to %s: %v\n", firstImagePath, baseOutputName, err)
		os.Exit(1)
	}
	fmt.Printf("Copied base image %s to %s\n", firstImagePath, baseOutputName)

	// 2. Process consecutive pairs for differences concurrently
	var wg sync.WaitGroup // Initialize WaitGroup

	var prevImage image.Image // Store the previously loaded image
	prevImage, err = loadImage(firstImagePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err) // loadImage already includes path info
		os.Exit(1)                   // Exit if the very first image fails to load
	}
	prevImagePath := firstImagePath

	for i := 1; i < len(inputFiles); i++ {
		currentImagePath := inputFiles[i]

		// Load current image sequentially
		currentImage, err := loadImage(currentImagePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintf(os.Stderr, "Skipping comparisons involving %s due to load error.\n", currentImagePath)
			prevImage = nil
			prevImagePath = ""
			continue
		}

		// If the previous image was loaded successfully, process the pair
		if prevImage != nil {
			wg.Add(1) // Increment counter before launching goroutine
			// processPair now handles generating the correct output path based on currentPath
			go processPair(&wg, prevImage, currentImage, prevImagePath, currentImagePath)
		} else {
			fmt.Fprintf(os.Stderr, "Skipping comparison for %s as previous image %s failed to load or process.\n", currentImagePath, prevImagePath)
		}

		// Update prevImage and prevImagePath for the *next* iteration's comparison
		prevImage = currentImage
		prevImagePath = currentImagePath
	}

	// Wait for all launched goroutines to complete
	fmt.Println("\nWaiting for image processing tasks to complete...")
	wg.Wait()

	fmt.Println("\nDiff processing complete.")
}

// runJoinMode handles the logic for reconstructing images sequentially.
func runJoinMode(inputFiles []string) {
	if len(inputFiles) < 2 {
		fmt.Fprintln(os.Stderr, "Error: -join mode requires at least two input images (base + diffs).")
		printUsage()
		os.Exit(1)
	}

	// Expecting files like: base.BASE.png, img2.DIFF.png, img3.DIFF.png ...
	baseImagePath := inputFiles[0]
	if !strings.Contains(filepath.Base(baseImagePath), ".BASE.") {
		fmt.Fprintf(os.Stderr, "Warning: First file %s for -join mode does not appear to be a .BASE file.\n", baseImagePath)
	}

	// Load the initial base image
	currentReconstructedImage, err := loadImage(baseImagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load base image %s: %v\n", baseImagePath, err)
		os.Exit(1)
	}

	// Save the first reconstructed image (which is just the base image)
	originalBaseName, err := generateOriginalFilename(baseImagePath) // Pass full path
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate original filename for base %s: %v\n", baseImagePath, err)
		os.Exit(1) // Cannot proceed without a valid name
	}

	outFileBase, err := os.Create(originalBaseName) // Use full output path
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output file %s: %v\n", originalBaseName, err)
		os.Exit(1)
	}
	// Encode requires an image.Image, loadImage returns one. Need to ensure PNG format?
	// Let's re-encode as PNG for consistency, though copying might be faster if format is known.
	err = png.Encode(outFileBase, currentReconstructedImage)
	outFileBase.Close() // Close immediately
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save reconstructed base image %s: %v\n", originalBaseName, err)
		os.Exit(1)
	}
	fmt.Printf("Saved reconstructed base image: %s\n", originalBaseName)

	// Process subsequent DIFF files sequentially
	for i := 1; i < len(inputFiles); i++ {
		diffImagePath := inputFiles[i]
		if !strings.Contains(filepath.Base(diffImagePath), ".DIFF.") {
			fmt.Fprintf(os.Stderr, "Warning: Input file %s for -join mode does not appear to be a .DIFF file.\n", diffImagePath)
		}
		fmt.Printf("\nApplying diff: %s\n", diffImagePath)

		diffImage, err := loadImage(diffImagePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load diff image %s: %v. Stopping reconstruction.\n", diffImagePath, err)
			os.Exit(1) // Cannot continue sequence if a diff is missing/corrupt
		}

		// Apply the diff to the last reconstructed image
		newReconstructedImage, err := applyDiff(currentReconstructedImage, diffImage)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to apply diff %s: %v. Stopping reconstruction.\n", diffImagePath, err)
			os.Exit(1)
		}

		// Generate output name for the *newly* reconstructed image
		originalDiffName, err := generateOriginalFilename(diffImagePath) // Pass full path
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate original filename for diff %s: %v\n", diffImagePath, err)
			os.Exit(1)
		}

		// Save the new reconstructed image
		outFileDiff, err := os.Create(originalDiffName) // Use full output path
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create output file %s: %v\n", originalDiffName, err)
			os.Exit(1)
		}
		err = png.Encode(outFileDiff, newReconstructedImage)
		outFileDiff.Close() // Close immediately
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save reconstructed image %s: %v\n", originalDiffName, err)
			os.Exit(1)
		}
		fmt.Printf("Saved reconstructed image: %s\n", originalDiffName)

		// Update the current reconstructed image for the next iteration
		currentReconstructedImage = newReconstructedImage
	}

	fmt.Println("\nJoin processing complete.")
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  differ -diff <image1> <image2> [image3 ...]")
	fmt.Println("    Generates base and difference images.")
	fmt.Println("    Output: <image1_name>.BASE.<ext>, <image2_name>.DIFF.<ext>, ...")
	fmt.Println("\n  differ -join <base_image> <diff_image2> [diff_image3 ...]")
	fmt.Println("    Reconstructs original images from base and difference files.")
	fmt.Println("    Input: image1.BASE.png image2.DIFF.png image3.DIFF.png ...")
	fmt.Println("    Output: image1.png, image2.png, image3.png, ...")
	fmt.Println("\nFlags:")
	flag.PrintDefaults() // Print default flag values (like -diff=false)
}

func main() {
	diffMode := flag.Bool("diff", false, "Generate difference files (BASE + DIFFs)")
	joinMode := flag.Bool("join", false, "Reconstruct images from BASE + DIFFs")

	flag.Parse()

	// Validate mode selection
	if (*diffMode && *joinMode) || (!*diffMode && !*joinMode) {
		fmt.Fprintln(os.Stderr, "Error: Please specify exactly one mode: -diff or -join")
		printUsage()
		os.Exit(1)
	}

	inputFiles := flag.Args()

	if *diffMode {
		fmt.Println("Mode: Diff")
		runDiffMode(inputFiles)
	} else if *joinMode {
		fmt.Println("Mode: Join")
		runJoinMode(inputFiles)
	}

	fmt.Println("\nProcessing complete.") // This might be redundant now
}
