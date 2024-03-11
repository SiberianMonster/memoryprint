// Handlers package contains endpoints handlers for the Photo Book Editor module.
//
// https://github.com/SiberianMonster/memoryprint/tree/development/internal/projecthandlers
package imagehandlers

import (
	"encoding/json"
	"log"
	"net/http"
	"image"
    "image/jpeg"
	"image/png"
    "os"
    "bytes"
	"strings"
	"strconv"
	"io/ioutil"
	"errors"

	"crypto/md5"
	"time"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	_ "github.com/lib/pq"
)

var err error
var resp map[string]string

type RBPostBody struct {
	mediaFile string `json:"mediaFile"`
}

type MyFile struct {
    *bytes.Reader
    mif myFileInfo
}
func (mf *MyFile) Close() error { return nil } // Noop, nothing to do

func (mf *MyFile) Readdir(count int) ([]os.FileInfo, error) {
    return nil, nil // We are not a directory but a single file
}

func (mf *MyFile) Stat() (os.FileInfo, error) {
    return nil, nil
}

type myFileInfo struct {
    name string
    data []byte
}

type balaResponse struct {
    Name string `json:"name"`
	Path string `json:"path"`
	Slug string `json:"slug"`
}

type ImageRespBody struct {
	Link string `json:"link"`
}

func GetToken(index int) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprint(time.Now()))))[0:index]
}

// JpegToPng converts a JPEG image to PNG format
func JpegToPng(imageBytes []byte) ([]byte, error) { 
    img, err := jpeg.Decode(bytes.NewReader(imageBytes))

    if err != nil {
        return nil, err
    }

    buf := new(bytes.Buffer)

    if err := png.Encode(buf, img); err != nil {
        return nil, err
    }

    return buf.Bytes(), nil
}

func saveImage(imgByte []byte, filename string) (string, error) {

    img, _, err := image.Decode(bytes.NewReader(imgByte))
    if err != nil {
        log.Printf("Image decoding error%s", err)
		return "", err
    }
    out, _ := os.Create(filename)
    defer out.Close()
	err = png.Encode(out, img)
	if err != nil {
		log.Printf("Image saving error%s", err)
	}
	return filename, err

}

func bucketUpload(img []byte, filename string, timewebToken string) error {

	filename, err = saveImage(img, filename)
	if err != nil {
		log.Printf("Failed to save file content %s", err)
		return err
	}
	log.Println("saved image")
	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)
	fw, err := writer.CreateFormFile(filename, filepath.Base(filename))
	if err != nil {
		log.Printf("Failed to create form file %s", err)
		return err
	}
	fd, err := os.Open(filename)
	if err != nil {
		log.Printf("Failed to open file %s", err)
		return err
	}
	defer fd.Close()
	_, err = io.Copy(fw, fd)
	if err != nil {
		log.Printf("Failed to copy file content %s", err)
		return err
	}

	writer.Close()

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.timeweb.cloud/api/v1/storages/buckets/225285/object-manager/upload?;path=photo/", form)
	if err != nil {
		log.Printf("Failed to create a request to bucket %s", err)
	}
	req.Header.Set("Authorization", "Bearer " + timewebToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
			log.Printf("Failed to make a request to bucket %s", err)
			return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
            log.Println(resp.StatusCode)
			err = errors.New("error uploading image to bucket")
            return err
    } else {
			log.Println("successful upload")
    }
	return nil
	
}

func removeBackground(imgByte []byte, filename string, balaToken string) ([]byte, error) {

	var bResp []balaResponse
	mf := &MyFile{
		Reader: bytes.NewReader(imgByte),
		mif: myFileInfo{
			name: "file.png",
			data: imgByte,
		},
	}
	
	var f http.File = mf
	fileContents, err := ioutil.ReadAll(f)
	if err != nil {
		log.Printf("Failed to read file contents %s", err)
	}

	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)
	fw, err := writer.CreateFormFile("mediaFile", "file.png")
	if err != nil {
		log.Printf("Failed to create form file %s", err)
	}
	fw.Write(fileContents)

	writer.Close()

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.ba-la.ru/api/remove", form)
	if err != nil {
		log.Printf("Failed to create a request to bucket %s", err)
	}
	req.Header.Set("api-key", balaToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to make a request to bala %s", err)
	}
	log.Println("response from bala")
	log.Println(resp.StatusCode)
	err = json.NewDecoder(resp.Body).Decode(&bResp)
	if err != nil {
		log.Printf("Failed to decode bala response %s", err)
	}
	log.Println(bResp)
	balaLinkContainer := bResp[1]
	balaURL := balaLinkContainer.Path
	balaURL = "https://ba-la.ru/" + balaURL
	log.Println(balaURL)
	defer resp.Body.Close()
	resp, err = http.Get(balaURL)
	if err != nil {
		log.Printf("Failed to read the file with removed background from bala %s", err)
	}
	defer resp.Body.Close()

	imgByte, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to get response from bala %s", err)
	}
	return imgByte, nil
}

func LoadImage(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]ImageRespBody)
	var imageObj models.UploadImage
	var filename string
	var rBody ImageRespBody
	//imageBytes = r.MultipartForm.File
	err := r.ParseMultipartForm(32 << 20) // maxMemory 32MB
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	imageObj.Extention = r.PostFormValue("extention")
	imageObj.RemoveBackground, _ = strconv.ParseBool(r.PostFormValue("remove_background"))
	file, _, err := r.FormFile("image")    
	defer file.Close()
	if err != nil {
		log.Printf("Error happened reading formfile. Err: %s", err)
		return 
	}

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		log.Printf("Error happened converting buffer to bytes slice. Err: %s", err)
		return 
	}
	imageObj.Image = buf.Bytes()
	log.Println("processed image")
	code := GetToken(10)
	filename = "./temp_photo/"+code+"_img.png"
	if imageObj.Extention == "jpeg" {

		// JpegToPng converts a JPEG image to PNG format
		imageObj.Image, err = JpegToPng(imageObj.Image)
		if err != nil {
			log.Printf("Error happened in jpeg to png converting. Err: %s", err)
			handlersfunc.HandleDecodeError(rw, err)
			return
		}
	}
	log.Println("converted image to jpg")
	if imageObj.RemoveBackground {

		imageObj.Image, err = removeBackground(imageObj.Image, filename, config.BalaToken)
		if err != nil {
			log.Printf("Error happened in removing image background. Err: %s", err)
			handlersfunc.HandleRemoveBackgroundError(rw)
			return
		}
	}
	log.Println("removed image background")
	
	err = bucketUpload(imageObj.Image, filename, config.TimewebToken)
	if err != nil {
		log.Printf("Error happened in uploading image to bucket. Err: %s", err)
		handlersfunc.HandleUploadImageError(rw)
		return
	}
	log.Println("uploaded image to bucket")
	trimmedName := strings.TrimLeft(filename, "./temp_photo/")
	
	rw.WriteHeader(http.StatusOK)
	rBody.Link = trimmedName
	resp["response"] = rBody
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}