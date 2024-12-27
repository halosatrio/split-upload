package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Metadata struct {
	Order    int    `json:"order"`
	FileId   string `json:"fileId"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
	FileSize int    `json:"fileSize"`
	FileName string `json:"fileName"`
}

func main() {
	r := gin.Default()

	r.MaxMultipartMemory = 8 << 20 // 8 MiB

	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.String(http.StatusBadRequest, "form data err: %s", err.Error())
			return
		}

		filename := filepath.Base(file.Filename)
		if err := c.SaveUploadedFile(file, "./uploads"+filename); err != nil {
			c.String(http.StatusBadRequest, "upload file error: %s", err.Error())
			return
		}
		c.JSON(200, gin.H{
			"status":  200,
			"message": "success single upload",
		})
	})

	r.POST("/split-upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.String(http.StatusBadRequest, "form data err: %s", err.Error())
			return
		}

		var metadata Metadata
		metadataJSON := c.Request.FormValue("metadata")
		err = json.Unmarshal([]byte(metadataJSON), &metadata)
		if err != nil {
			c.String(http.StatusBadRequest, "error unmarshal metadata: %s", err.Error())
			return
		}

		if err := c.SaveUploadedFile(file, fmt.Sprintf("./uploads/temp/%v_%v", metadata.Order, metadata.FileId)); err != nil {
			c.String(http.StatusBadRequest, "error upload chunk file: %s", err.Error())
			return
		}

		if metadata.FileSize == metadata.Limit {
			chunks, err := filepath.Glob(filepath.Join("./uploads/temp", fmt.Sprintf("*_%s", metadata.FileId)))
			if err != nil {
				c.String(http.StatusBadRequest, "error finding chunk file: %s", err.Error())
				return
			}

			sort.Slice(chunks, func(i, j int) bool {
				orderI, _ := strconv.Atoi(string(filepath.Base(chunks[i])[0]))
				orderJ, _ := strconv.Atoi(string(filepath.Base(chunks[j])[0]))

				return orderI < orderJ
			})

			finalPath := filepath.Join("./uploads", fmt.Sprintf("merged_%s", metadata.FileName))
			finalFile, err := os.Create(finalPath)
			if err != nil {
				c.String(http.StatusBadRequest, "error merging file: %s", err.Error())
				return
			}
			defer finalFile.Close()

			for _, chunk := range chunks {
				chunkFile, err := os.Open(chunk)
				if err != nil {
					c.String(http.StatusBadRequest, "error open chunk file: %s", err.Error())
					return
				}

				_, err = io.Copy(finalFile, chunkFile)
				chunkFile.Close()

				if err != nil {
					c.String(http.StatusBadRequest, "error merging chunk file: %s", err.Error())
					return
				}
			}

			// remove chunks
			for _, chunk := range chunks {
				os.Remove(chunk)
			}

			log.Println("chunk upload success, merge chunk files")
		}

		c.SaveUploadedFile(file, "./uploads/temp/")
		c.JSON(200, gin.H{
			"status":  200,
			"message": "success split upload",
		})
	})

	r.Run(":8080")
}
