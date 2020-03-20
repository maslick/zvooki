package main

import (
	"fmt"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	rest := RestController{&Service{}}
	rest.Start()
}

///////////////////////
// Service
//////////////////////
type IService interface {
	GetAudioFile(id string) []byte
	GetAudioRange(id string, start, end int) []byte
	GetContentType(id string) string
	GetFileSize(id string) int
}

type Service struct{}

func (s *Service) GetAudioRange(id string, start, end int) []byte {
	f, err := os.Open("files/" + id)
	chk(err)
	defer f.Close()

	result := make([]byte, end-start+1)
	_, err = f.ReadAt(result, int64(start))
	chk(err)
	return result
}

func (s *Service) GetAudioFile(id string) []byte {
	fileBytes, err := ioutil.ReadFile("files/" + id)
	chk(err)
	return fileBytes
}

func (s *Service) GetContentType(id string) string {
	if strings.HasSuffix(id, ".mp3") {
		return "audio/mp3"
	}
	if strings.HasSuffix(id, ".mp4") {
		return "video/mp4"
	}
	return ""
}

func (s *Service) GetFileSize(id string) int {
	fileBytes, err := ioutil.ReadFile("files/" + id)
	chk(err)
	return len(fileBytes)
}

///////////////////////
// Controller
//////////////////////
type RestController struct {
	Service IService
}

func (api *RestController) Start() {
	port := getPort()
	fmt.Println("Starting server on port:", strings.Split(port, ":")[1])
	log.Fatal(http.ListenAndServe(port, api.initRouter()))
}

func (api *RestController) initRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(static.Serve("/", static.LocalFile("./static", true)))

	public := router.Group("/")
	public.GET("/media/:id", api.soundHandler)
	return router
}

func (api *RestController) soundHandler(c *gin.Context) {
	id := c.Param("id")
	rang := c.Request.Header.Get("range")

	fileSize := api.Service.GetFileSize(id)
	if len(rang) > 0 {
		parts := strings.Split(strings.ReplaceAll(rang, "bytes=", ""), "-")
		start, _ := strconv.Atoi(parts[0])
		end, _ := strconv.Atoi(parts[1])
		if end == 0 {
			end = fileSize - 1
		}

		if start >= fileSize {
			http.Error(c.Writer, fmt.Sprintf("Requested range not satisfiable\n%d >= %d", start, fileSize), 416)
		}

		chunkSize := (end - start) + 1
		c.Status(206)
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		c.Header("Accept-Ranges", "bytes")
		c.Header("Content-Length", fmt.Sprintf("%d", chunkSize))
		c.Header("Content-Type", api.Service.GetContentType(id))
		c.Writer.Write(api.Service.GetAudioRange(id, start, end))
		return
	}

	c.Status(200)
	c.Header("Content-Type", api.Service.GetContentType(id))
	c.Header("Content-Length", fmt.Sprintf("%d", fileSize))
	c.Writer.Write(api.Service.GetAudioFile(id))
}

///////////////////////
// Helper methods
//////////////////////
func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func getPort() string {
	var port = getEnv("PORT", "8080")
	return ":" + port
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
