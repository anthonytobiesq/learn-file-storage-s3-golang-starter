package media

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type ProbeOutput struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func GetVideoAspectRatio(filePath string) (string, error) {
	// use exec.Command to run the same ffprobe command as above.
	//In this case, the command is ffprobe, and the arguments are -v, error,
	//-print_format, json, -show_streams, and the file path.
	cmd := exec.Command(
		"ffprobe",
		"-v",
		"error",
		"-print_format",
		"json",
		"-show_streams",
		filePath)
	//capture the output
	var b bytes.Buffer
	cmd.Stdout = &b
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed running ffprobe: %v", err)
	}

	//parse the JSON output
	var output ProbeOutput
	err = json.Unmarshal(b.Bytes(), &output)
	if err != nil {
		return "", fmt.Errorf("failed parsing ffprobe output: %v", err)
	}

	if len(output.Streams) == 0 {
		return "", fmt.Errorf("no streams found in ffprobe output")
	}

	//calculate the aspect ratio
	aspectRatio := evaluateAspectRatio(output.Streams[0].Width, output.Streams[0].Height)

	return aspectRatio, nil

}

func evaluateAspectRatio(w, h int) string {
	if h == 0 {
		return "other"
	}

	ratio := float64(w) / float64(h)

	// We use a small "epsilon" (0.1) to account for slight variations
	switch {
	case ratio > 1.6 && ratio < 1.9:
		return "16:9"
	case ratio > 0.5 && ratio < 0.65:
		return "9:16"
	default:
		return "other"
	}
}

func ProcessVideoForFastStart(filePath string) (string, error) {
	//Create a new string for the output file path. I just appended .processing to the input file (which should be the path to the temp file on disk)
	outputFilePath := filePath + ".processing"

	//Create a new exec.Cmd using exec.Command
	//The command is ffmpeg and the arguments are -i, the input file path, -c, copy, -movflags, faststart, -f, mp4 and the output file path.
	cmd := exec.Command(
		"ffmpeg",
		"-i",
		filePath,
		"-c",
		"copy",
		"-movflags",
		"faststart",
		"-f",
		"mp4",
		outputFilePath,
	)

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed running ffmpeg: %v", err)
	}

	return outputFilePath, nil
}
