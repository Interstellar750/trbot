package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func echoSticker(filePath string) (*io.PipeReader) {
	fmt.Printf("https://api.telegram.org/file/bot%s/%s", os.Getenv("TELEGRAM_BOT_TOKEN"), filePath)
	resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", os.Getenv("TELEGRAM_BOT_TOKEN"), filePath))
	if err != nil { log.Printf("error downloading file: %v", err) }
	// defer resp.Body.Close()
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		_, err := io.Copy(writer, resp.Body)
		if err != nil {
			fmt.Println("Error copying to pipe:", err)
		}
	}()

	return reader
}
