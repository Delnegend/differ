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

// generateOutputFilename creates output path with specified suffix before file extension.
func generateOutputFilename(inputPath, suffix string) (string, error) {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	baseName := filepath.Base(inputPath)
	if len(ext) > 0 {
		baseName = strings.TrimSuffix(baseName, ext)
	} else {
		fmt.Fprintf(os.Stderr, "Warning: Input file %s has no extension, assuming %s for output.\n", inputPath, ext)
	}
	if suffix != "" && !strings.HasPrefix(suffix, ".") {
		suffix = "." + suffix
	}

	outputBaseName := fmt.Sprintf("%s%s%s", baseName, suffix, ext)
	return filepath.Join(dir, outputBaseName), nil
}

// generateOriginalFilename strips .BASE or .DIFF suffixes to reconstruct the original name.
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
	return destFile.Sync()
}

// createDiffImage compares two images and returns an RGBA image holding the differences.
func createDiffImage(baseImg, currentImg image.Image) (*image.RGBA, int) {
	bounds := baseImg.Bounds()
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

// processPair compares two images and writes differences as a DIFF image.
func processPair(wg *sync.WaitGroup, prevImg image.Image, currentImg image.Image, prevPath, currentPath string) {
	defer wg.Done()

	fmt.Printf("Processing pair: %s vs %s\n", prevPath, currentPath)

	bounds1 := prevImg.Bounds()
	bounds2 := currentImg.Bounds()
	if bounds1 != bounds2 {
		fmt.Fprintf(os.Stderr, "Error: Image dimensions do not match (%s vs %s) for pair (%s, %s). Skipping.\n",
			bounds1, bounds2, prevPath, currentPath)
		return
	}

	diffImg, diffPixels := createDiffImage(prevImg, currentImg)
	fmt.Printf("Found %d different pixels between %s and %s.\n", diffPixels, prevPath, currentPath)

	diffOutputName, err := generateOutputFilename(currentPath, "DIFF")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating diff filename for %s: %v. Skipping save.\n", currentPath, err)
		return
	}

	outFile, err := os.Create(diffOutputName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output file %s: %v\n", diffOutputName, err)
		return
	}
	defer outFile.Close()

	err = png.Encode(outFile, diffImg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode difference image to PNG %s: %v\n", diffOutputName, err)
	} else {
		fmt.Printf("Difference image saved to %s\n", diffOutputName)
	}
}

// applyDiff reconstructs an image by applying diff pixels over the base image.
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

// runDiffMode generates BASE and DIFF images concurrently.
func runDiffMode(inputFiles []string) {
	if len(inputFiles) < 2 {
		fmt.Fprintln(os.Stderr, "Error: -diff mode requires at least two input images.")
		printUsage()
		os.Exit(1)
	}

	firstImagePath := inputFiles[0]
	baseOutputName, err := generateOutputFilename(firstImagePath, "BASE")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating base filename for %s: %v\n", firstImagePath, err)
		os.Exit(1)
	}
	err = copyFile(firstImagePath, baseOutputName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying base image %s to %s: %v\n", firstImagePath, baseOutputName, err)
		os.Exit(1)
	}
	fmt.Printf("Copied base image %s to %s\n", firstImagePath, baseOutputName)

	var wg sync.WaitGroup
	var prevImage image.Image
	prevImage, err = loadImage(firstImagePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	prevImagePath := firstImagePath

	for i := 1; i < len(inputFiles); i++ {
		currentImagePath := inputFiles[i]

		currentImage, err := loadImage(currentImagePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintf(os.Stderr, "Skipping comparisons involving %s due to load error.\n", currentImagePath)
			prevImage = nil
			prevImagePath = ""
			continue
		}

		if prevImage != nil {
			wg.Add(1)
			go processPair(&wg, prevImage, currentImage, prevImagePath, currentImagePath)
		} else {
			fmt.Fprintf(os.Stderr, "Skipping comparison for %s as previous image %s failed to load or process.\n", currentImagePath, prevImagePath)
		}

		prevImage = currentImage
		prevImagePath = currentImagePath
	}

	fmt.Println("\nWaiting for image processing tasks to complete...")
	wg.Wait()

	fmt.Println("\nDiff processing complete.")
}

// runJoinMode reconstructs original images from BASE and DIFF files sequentially.
func runJoinMode(inputFiles []string) {
	if len(inputFiles) < 2 {
		fmt.Fprintln(os.Stderr, "Error: -join mode requires at least two input images (base + diffs).")
		printUsage()
		os.Exit(1)
	}

	baseImagePath := inputFiles[0]
	if !strings.Contains(filepath.Base(baseImagePath), ".BASE.") {
		fmt.Fprintf(os.Stderr, "Warning: First file %s for -join mode does not appear to be a .BASE file.\n", baseImagePath)
	}

	currentReconstructedImage, err := loadImage(baseImagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load base image %s: %v\n", baseImagePath, err)
		os.Exit(1)
	}

	originalBaseName, err := generateOriginalFilename(baseImagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate original filename for base %s: %v\n", baseImagePath, err)
		os.Exit(1)
	}

	outFileBase, err := os.Create(originalBaseName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output file %s: %v\n", originalBaseName, err)
		os.Exit(1)
	}
	err = png.Encode(outFileBase, currentReconstructedImage)
	outFileBase.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save reconstructed base image %s: %v\n", originalBaseName, err)
		os.Exit(1)
	}
	fmt.Printf("Saved reconstructed base image: %s\n", originalBaseName)

	for i := 1; i < len(inputFiles); i++ {
		diffImagePath := inputFiles[i]
		if !strings.Contains(filepath.Base(diffImagePath), ".DIFF.") {
			fmt.Fprintf(os.Stderr, "Warning: Input file %s for -join mode does not appear to be a .DIFF file.\n", diffImagePath)
		}
		fmt.Printf("\nApplying diff: %s\n", diffImagePath)

		diffImage, err := loadImage(diffImagePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load diff image %s: %v. Stopping reconstruction.\n", diffImagePath, err)
			os.Exit(1)
		}

		newReconstructedImage, err := applyDiff(currentReconstructedImage, diffImage)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to apply diff %s: %v. Stopping reconstruction.\n", diffImagePath, err)
			os.Exit(1)
		}

		originalDiffName, err := generateOriginalFilename(diffImagePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate original filename for diff %s: %v\n", diffImagePath, err)
			os.Exit(1)
		}

		outFileDiff, err := os.Create(originalDiffName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create output file %s: %v\n", originalDiffName, err)
			os.Exit(1)
		}
		err = png.Encode(outFileDiff, newReconstructedImage)
		outFileDiff.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save reconstructed image %s: %v\n", originalDiffName, err)
			os.Exit(1)
		}
		fmt.Printf("Saved reconstructed image: %s\n", originalDiffName)

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
	flag.PrintDefaults()
}

func main() {
	diffMode := flag.Bool("diff", false, "Generate difference files (BASE + DIFFs)")
	joinMode := flag.Bool("join", false, "Reconstruct images from BASE + DIFFs")

	flag.Parse()

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

	fmt.Println("\nProcessing complete.")
}
