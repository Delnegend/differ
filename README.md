# Differ - Image Differencing Tool

`differ` is a command-line utility designed to reduce the storage space required for sequences of similar images. It achieves this by storing the first image as a full "base" image and subsequent images as "difference" (diff) images, containing only the pixels that changed compared to the previous one. This concept is similar to inter-frame compression techniques used in video codecs like H.264, H.265, AV1, and VP9.

When needed, the original sequence of images can be perfectly reconstructed by starting with the base image and sequentially applying the diff images.

## Features

*   **Diff Mode (`-diff`):** Generates a `.BASE` copy of the first image and `.DIFF` files for subsequent images.
*   **Join Mode (`-join`):** Reconstructs the original image sequence from a `.BASE` file and subsequent `.DIFF` files.

## Use Cases

*   Storing sequences of screenshots where only small portions change (e.g., UI testing, tutorials).
*   Archiving versions of digital artwork or designs.
*   Any scenario involving multiple, similar images where storage efficiency is desired.

## Usage

The tool operates in one of two modes: `-diff` or `-join`.

### Generating Diff Files (`-diff`)

Provide the sequence of original images as arguments. The first image will be copied with a `.BASE` suffix, and subsequent images will generate diff files with a `.DIFF` suffix relative to the *previous* image in the sequence.

```bash
# Example: Process three images
differ -diff frame01.png frame02.png frame03.png

# Output:
# frame01.BASE.png  (Copy of frame01.png)
# frame02.DIFF.png  (Differences between frame01.png and frame02.png)
# frame03.DIFF.png  (Differences between frame02.png and frame03.png)
```

### Reconstructing Original Images (`-join`)

Provide the `.BASE` file followed by the sequence of `.DIFF` files in the correct order. The tool will reconstruct and save the original images.

```bash
# Example: Reconstruct the three images from the previous example
differ -join frame01.BASE.png frame02.DIFF.png frame03.DIFF.png

# Output:
# frame01.png (Reconstructed from frame01.BASE.png)
# frame02.png (Reconstructed from frame01.BASE.png + frame02.DIFF.png)
# frame03.png (Reconstructed from frame02.png + frame03.DIFF.png)
```

## Releases

Check the [Releases](https://github.com/Delnegend/differ/releases) page for pre-built binaries.

## Building

Require the latest version of Go to build the tool.

1.  Clone the repository or download the source code.
2.  Navigate to the project directory in your terminal.
3.  Run the build command:
    ```bash
    go build .
    ```
4.  This will create the `differ` (or `differ.exe` on Windows) executable in the current directory.


## License
Licensed under either of

- Apache License, Version 2.0 ([LICENSE-Apache](./LICENSE-Apache) or [apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0))
- MIT license ([LICENSE-MIT](./LICENSE-MIT) or [opensource.org/licenses/MIT](https://opensource.org/licenses/MIT))
at your option.

## Contribution
Unless you explicitly state otherwise, any contribution intentionally submitted for inclusion in the work by you, as defined in the Apache-2.0 license, shall be dual licensed as above, without any additional terms or conditions.