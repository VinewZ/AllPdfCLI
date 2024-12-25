# AudioBook-Generator

This Go project is designed to convert a PDF book into an audio format using [AllTalk TTS](https://github.com/erew123/alltalk_tts/tree/alltalkbeta).
The program extracts text from the PDF, generates audio for each sentence using a TTS (Text-to-Speech) service, and concatenates the audio into chapters.

## Features

- Converts PDF to text.
- Generates audio files for each chapter.
- Supports customization of language and delay between sentences.
- Handles temporary directories for processing and final outputs.
- Allows easy chapter audio concatenation with silence in between.

## Requirements

- Go 1.18+.
- Dependencies:
  - `github.com/gen2brain/go-fitz`: PDF processing library.
  - `github.com/iFaceless/godub`: Audio processing library.
  - `github.com/jdkato/prose/v2`: Natural language processing library.
  - `github.com/erew123/alltalk_tts/alltalkbeta`: TTS service.

## Usage

1. Clone this repository:

```bash
git clone https://github.com/VinewZ/Audiobook-Generator.git
cd Audiobook-Generator
```

## Install dependencies:

```bash
go mod tidy
```

## Run

```bash
go run main.go <pdf_path> <book_title> <delay_in_ms> <language>

    <pdf_path>: Path to the PDF file you want to convert.
    <book_title>: Title of the book (will be used as a folder name).
    <delay_in_ms>: Delay in milliseconds between sentences in the audio.
    <language>: Language for the TTS service (e.g., "en").
```

Example:

```bash
    go run main.go /path/to/book.pdf "My Book" 700 "en"
```


## File Structure

    ./tmp: Temporary directory for processing files.
    ./tmp/<book_title>/txts: Contains the extracted text files from the PDF.
    ./tmp/<book_title>/audios: Contains individual audio files for each sentence.
    ./tmp/<book_title>/audios/final: Contains the concatenated audio files for each chapter.

## How It Works

    Initialize: The program checks the input arguments and sets up temporary directories.
    Create Temporary PDF: The original PDF file is copied to a temporary location for processing.
    Text Extraction: The program uses go-fitz to extract text from the PDF and save it into text files by chapters.
    Audio Generation: For each chapter, the program sends text to a TTS API and receives audio files.
    Concatenation: The individual sentence audio files are concatenated into full chapter audio files, with a silent gap (delay) between sentences.

## Example Output

After running the program, you'll find the generated audio files in the following directory:

./tmp/<book_title>/audios/final

Each chapter will have a corresponding .wav file.

## Notes

    The TTS service is expected to be running on http://127.0.0.1:7851/api/tts-generate. Make sure to have a TTS API service available for generating audio.
    The program creates temporary directories (./tmp) for storing intermediate files. These can be cleaned up after the process completes.
