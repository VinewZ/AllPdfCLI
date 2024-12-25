package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/go-fitz"
	"github.com/iFaceless/godub"
	"github.com/jdkato/prose/v2"
)

type model struct {
	pdfOriginalPath string
	pdfDirTmpPath   string
	pdfFileTmpPath  string
	bookTitle       string
	delay           float64
	txtsDirPath     string
	audiosDirPath   string
	language        string
}

const (
  TmpDir = "./tmp"
  Checkmark = '\u2713'
)

func main() {

	m := model{}

	m.initialize(os.Args)
	m.createTmpPdf()
	m.extractText()
	m.generateAudios()
}

func (m *model) initialize(args []string) {
	if len(args) < 5 {
    logError(fmt.Errorf("Invalid number of arguments"), "\nUsage: <program> <pdf_path> <book_title> <delay> <language>")
	}

	m.pdfOriginalPath = args[1]
	m.bookTitle = sanitizeTitle(args[2])
	m.delay = parseFloatOrExit(args[3])
	m.language = args[4]

	m.pdfDirTmpPath = path.Join(TmpDir, m.bookTitle)
	m.pdfFileTmpPath = filepath.Join(m.pdfDirTmpPath, fmt.Sprintf("%s.pdf", m.bookTitle))
	m.txtsDirPath = path.Join(m.pdfDirTmpPath, "txts")
	m.audiosDirPath = path.Join(m.pdfDirTmpPath, "audios")

	createDirectories(m.pdfDirTmpPath, m.txtsDirPath, m.audiosDirPath)
}

func (m model) createTmpPdf() {
	now := time.Now()
	fmt.Printf("Copying file")
	srcF, err := os.ReadFile(m.pdfOriginalPath)
  logError(err, "reading source file")

	err = os.WriteFile(m.pdfFileTmpPath, []byte(srcF), os.ModePerm)
  logError(err, "writing tmp pdf file")

	elapsed := time.Since(now)
	fmt.Printf("\rFile copied %c - %.3fs\n", Checkmark, elapsed.Seconds())
}

func (m *model) extractText() {
	now := time.Now()
	file, err := fitz.New(m.pdfFileTmpPath)
	if err != nil {
		fmt.Printf("Error while opening file with fitz: %v\n", err)
	}
	defer file.Close()

	toc, err := file.ToC()
	if err != nil {
		fmt.Printf("Error while getting TOC: %v\n", err)
	}

	for idx := 0; idx < len(toc)-1; idx++ {
		txtFilePath := fmt.Sprintf("%s/%d.txt", m.txtsDirPath, toc[idx].Page)
		txtFile, err := os.Create(txtFilePath)
		if err != nil {
			fmt.Printf("Error while creating txt file: %v\n", err)
		}
		defer txtFile.Close()

		for pg := toc[idx].Page; pg < toc[idx+1].Page; pg++ {
			text, err := file.Text(pg)
			if err != nil {
				fmt.Printf("Error EXTRACTING text from page %d: %v\n", pg, err)
			}

			_, err = txtFile.WriteString(text)
			if err != nil {
				fmt.Printf("Error WRITING text from page %d: %v\n", pg, err)
			}
			fmt.Printf("\rExtracting Chapter: %d - Page: %d of %d", idx, pg, toc[idx+1].Page)
		}
	}

	elapsed := time.Since(now)
	fmt.Printf("\rText extracted %c - %.3fs %-*s \n", Checkmark, elapsed.Seconds(), 50, "")
}

func (m *model) generateAudios() {
	now := time.Now()
	fmt.Printf("Generating audios")

	files, err := os.ReadDir(m.txtsDirPath)
	if err != nil {
		fmt.Printf("Error reading txts dir: %v\n", err)
	}

	sort.Sort(ByNumber(files, func(entry os.DirEntry) int {
		num, _ := strconv.Atoi(entry.Name()[:len(entry.Name())-4])
		return num
	}))

	var audioCount = 0
	for fIdx, f := range files {
		fo, err := os.ReadFile(path.Join(m.txtsDirPath, f.Name()))
		if err != nil {
			fmt.Printf("Error opening file %s: %v", f.Name(), err)
		}

		doc, err := prose.NewDocument(string(fo))
    logError(err, fmt.Sprintf("creating document for chapter %d", fIdx+1))

		for stcIdx, stc := range doc.Sentences() {
			if stc.Text == "" || stc.Text == " " {
				continue
			}
			stcNow := time.Now()
			audioCount++
			formData := url.Values{
				"text_input":            {stc.Text},
				"text_filtering":        {"standard"},
				"character_voice_gen":   {"female_01.wav"},
				"narrator_enabled":      {"false"},
				"narrator_voice_gen":    {"male_01.wav"},
				"text_not_inside":       {"character"},
				"language":              {m.language},
				"output_file_name":      {fmt.Sprintf("%s_%04d", m.bookTitle, audioCount)},
				"output_file_timestamp": {"false"},
				"autoplay":              {"false"},
				"autoplay_volume":       {"0.1"},
			}
			postSent, err := http.PostForm(
				"http://127.0.0.1:7851/api/tts-generate",
				formData,
			)
      logError(err, "posting form to API")
			defer postSent.Body.Close()

			bd, err := io.ReadAll(postSent.Body)
      logError(err, "reading response body")

			type TTSResponse struct {
				Status         string `json:"status"`
				OutputFilePath string `json:"output_file_path"`
				OutputFileURL  string `json:"output_file_url"`
				OutputCacheURL string `json:"output_cache_url"`
			}

			var resJson TTSResponse

			err = json.Unmarshal(bd, &resJson)
      logError(err, "unmarshaling response")

			m.saveAudioFile(resJson.OutputFilePath, fIdx+1)
			fElapsed := time.Since(now)
			stcElapsed := time.Since(stcNow)
			fmt.Printf("\rChapter: %d of %d - %.2fh | Sentence: %d of %d - %.3fs",
        fIdx+1, len(files), fElapsed.Hours(), stcIdx+1, len(doc.Sentences()), stcElapsed.Seconds())
		}
		m.concatenateAudios(fIdx)
	}

	elapsed := time.Since(now)
	fmt.Printf("\rAudios generated %c - %.3fs %-*s \n", Checkmark, elapsed.Seconds(), 50, "")
}

func (m *model) saveAudioFile(audioSrc string, chp int) {
	audioName := filepath.Base(audioSrc)

	chapDirPath := fmt.Sprintf("%s/%d", m.audiosDirPath, chp)
	err := os.MkdirAll(chapDirPath, os.ModePerm)
  logError(err, fmt.Sprintf("creating chapter %d directory", chp))

	err = os.Rename(audioSrc, path.Join(chapDirPath, audioName))
  logError(err, fmt.Sprintf("moving audio %s to chapter %d", audioName, chp))
}

func (m *model) concatenateAudios(chp int) {
	chpPath := fmt.Sprintf("%s/%d", m.audiosDirPath, chp+1)
	audios, err := os.ReadDir(chpPath)
  logError(err, fmt.Sprintf("reading audios directory for chapter %d", chp+1))

	segment, err := godub.NewLoader().Load(path.Join(chpPath, audios[0].Name()))
  logError(err, "loading first audio")

	silence := m.createSilentAudio(m.delay)

	if len(audios) > 1 {
		for i := 1; i < len(audios); i++ {
			fmt.Printf("\rAppending audio %d / %d %-*s", i, len(audios), 50, "")
			audioPath := path.Join(chpPath, audios[i].Name())
			newSeg, err := godub.NewLoader().Load(audioPath)
      logError(err, fmt.Sprintf("loading audio %s", audios[i].Name()))

			silenceDub, err := godub.NewLoader().Load(silence)
      logError(err, "loading silence audio")

			segment, err = segment.Append(silenceDub, newSeg)
      logError(err, "appending audio")
		}
	} 

	outPath := path.Join(m.pdfDirTmpPath, "final", fmt.Sprintf("%d-%s.wav", chp+1, m.bookTitle))
  err = os.MkdirAll(path.Dir(outPath), os.ModePerm)
  logError(err, "creating final directory")

	err = godub.NewExporter(outPath).WithDstFormat("wav").WithBitRate(128).Export(segment)
  logError(err, "exporting final audio")
}

func (m *model) createSilentAudio(duration float64) string {
	segment, err := godub.NewSilentAudioSegment(duration, 24000)
  logError(err, "creating silent audio")

	outPath := path.Join("./tmp", m.bookTitle, "audios/silence.wav")
	err = godub.NewExporter(outPath).WithDstFormat("wav").WithBitRate(128).Export(segment)
  logError(err, "exporting silent audio")

	return outPath
}

func createSilentAudio(fileName string, duration int) string {
  segment, err := godub.NewSilentAudioSegment(float64(duration), 24000)
  if err != nil {
    log.Fatalf("Error while creating silent audio: %v", err)
  }

  outPath := path.Join("./tmp", fileName, "audios/silence.wav")
  err = godub.NewExporter(outPath).WithDstFormat("wav").WithBitRate(128).Export(segment)
  if err != nil {
    log.Fatalf("Error while exporting silent audio: %v", err)
  }

  return outPath
}

func createDirectories(dirs ...string) {
	for _, dir := range dirs {
		err := os.MkdirAll(dir, os.ModePerm)
		logError(err, fmt.Sprintf("creating directory %s", dir))
	}
}

func parseFloatOrExit(value string) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
  logError(err, "parsing float")
	return parsed
}

func sanitizeTitle(title string) string {
	return strings.ReplaceAll(strings.TrimSpace(title), " ", "_")
}

func logError(err error, message string) {
	if err != nil {
		log.Fatalf("Error %s: %v", message, err)
	}
}
