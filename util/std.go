package util

import (
	"bufio"
	"fmt"
	log "github.com/pion/ion-log"
	"io"
	"os/exec"
)

func GetStdPipes(ffmpeg *exec.Cmd) (io.WriteCloser, io.ReadCloser, error) {
	ffmpegIn, err := ffmpeg.StdinPipe()
	if err != nil {
		log.Errorf("Cannot get ffmpeg.StdinPipe(): %+v", err)
		return nil, nil, err
	}
	ffmpegOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		log.Errorf("Cannot get ffmpeg.StdoutPipe(): %+v", err)
		return nil, nil, err
	}
	ffmpegErr, err := ffmpeg.StderrPipe()
	if err != nil {
		log.Errorf("Cannot get ffmpeg.StderrPipe(): %+v", err)
		return nil, nil, err
	}

	if err := ffmpeg.Start(); err != nil {
		log.Errorf("Cannot Start ffmpeg: %+v", err)
		return nil, nil, err
	}

	go func(ffmpegErr io.ReadCloser) {
		scanner := bufio.NewScanner(ffmpegErr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}(ffmpegErr)
	return ffmpegIn, ffmpegOut, nil
}
