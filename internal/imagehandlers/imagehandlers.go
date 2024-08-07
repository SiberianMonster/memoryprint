// Handlers package contains endpoints handlers for the Photo Book Editor module.
//
// https://github.com/SiberianMonster/memoryprint/tree/development/internal/projecthandlers
package imagehandlers

import (
	"encoding/json"
	"context"
	"log"
	"net/http"
	"image"
	"image/png"
	"image/jpeg"
    "os"
    "bytes"
	"strings"
	"strconv"
	"io/ioutil"
	"errors"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/gorilla/mux"

	"crypto/md5"
	"crypto/tls"
	"time"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	"github.com/SiberianMonster/memoryprint/internal/projectstorage"
	_ "github.com/lib/pq"
)
const BINARY = "/usr/bin/inkscape"
var imageContentTypes = map[string]string{
    "png":  "image/png",
    "jpg":  "image/jpeg",
    "jpeg": "image/jpeg",
}
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
	result :=  fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprint(time.Now()))))[0:index]
	if len(result) < 10 {
		result = result + "x"
	}
	return result
}


func DownloadFile(filepath string, url string) error {

    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    out, err := os.Create(filepath)
    if err != nil {
        return err
    }
    defer out.Close()

    //_, err = io.Copy(out, resp.Body)
	//out.Flush()
	out.ReadFrom(resp.Body)
    return err
}

func saveImage(imgByte []byte, filename string) (string, error) {

	
	if strings.Contains(filename, "svg") {
		dst, err := os.Create(filename)
        if err != nil {
            log.Println("error creating file", err)
            return "", err
        }
        defer dst.Close()
		err = os.WriteFile(filename, imgByte, 0644)
		if err != nil { 
			log.Printf("SVG decoding error%s", err)
			return "", err
		}
	} else if strings.Contains(filename, "png") {
		log.Printf("Started image processing")
		img, _, err := image.Decode(bytes.NewReader(imgByte))
		if err != nil {
			log.Printf("Image decoding error%s", err)
			return "", err
		}
		log.Printf("Decoded")
		out, _ := os.Create(filename)
		defer out.Close()
		log.Printf("Created file")
		err = png.Encode(out, img)
		if err != nil {
			log.Printf("Image saving error%s", err)
			return "", err
		}
		log.Printf("Encoded")
	} else if strings.Contains(filename, "jpeg") {
		img, err := jpeg.Decode(bytes.NewReader(imgByte))
		if err != nil {
			log.Printf("Image decoding error%s", err)
			return "", err
		}
		out, _ := os.Create(filename)
		defer out.Close()
		err = jpeg.Encode(out, img, &jpeg.Options{100})
		if err != nil {
			log.Printf("Image saving error%s", err)
			return "", err
		}
	}

	return filename, err

}

func bucketUpload(img []byte, filename string, timewebToken string) error {

	filename, err = saveImage(img, filename)
	log.Println("error")
	if filename == "" {
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

func bucketPdfUpload(filename string, timewebToken string) error {

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
	tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }

	client := &http.Client{Transport: tr}
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
			err = errors.New("error uploading pdf to bucket")
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
	log.Println(imageObj.Extention)
	_, found := imageContentTypes[imageObj.Extention]
    if !found {
        handlersfunc.HandleWrongImageFormatError(rw)
		return
    }
	
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
	if len(imageObj.Image) == 0 {
		log.Printf("Error happened in reading image. Empty bytes slice. Err: %s", err)
		handlersfunc.HandleMissingImageDataError(rw)
		return
	}
	log.Println("processed image")
	code := GetToken(10)
	filename = "./temp_photo/"+code+"_img."+imageObj.Extention
	log.Println(filename)
	
	if imageObj.RemoveBackground && imageObj.Extention != "svg" {

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
	trimmedName := strings.Replace(filename, "./temp_photo/", "", 1)
	log.Println(trimmedName)
	//if len(trimmedName) < 18 {
	//	log.Printf("Error happened in uploading image to bucket. Name is too short Err: ")
	//	handlersfunc.HandleUploadImageError(rw)
	//	return
	//}
	err = os.Remove(filename) 
    if err != nil { 
        log.Printf("Error happened in removing image after bucket upload. Err: %s", err)
		handlersfunc.HandleUploadImageError(rw)
		return
    } 
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


func CreatePDFVisualization(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]ImageRespBody)
	var rBody ImageRespBody
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	print(projectID)
	defer r.Body.Close()

	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Create project visualization %d for user %d",projectID, userID)

	userCheck := projectstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw)
		return
	}
	
	pages, err := projectstorage.RetrieveProjectPages(ctx, config.DB, projectID, false)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	var pdfName string
	code := GetToken(10)
	pdfName = "pdflink_" + strconv.Itoa(aByteToInt) + "_" + code + ".pdf"
	var pagesImages []string
	//dir, _ := ioutil.TempDir("", strconv.Itoa(aByteToInt))
	if err != nil {
		log.Printf("Error happened in creating temp dir for the images. Err: %s", err)
		handlersfunc.HandleUploadImageError(rw)
	}
	//defer os.RemoveAll(dir)
	for _, page := range pages {

		strCreatingImageLink := *page.CreatingImageLink
		imageURL := config.ImageHost+strCreatingImageLink
		localPath := "./temp_pdf/" + strCreatingImageLink
		log.Println(localPath)
		err = DownloadFile(localPath, imageURL) 
		if err != nil {
			log.Printf("Error happened in loading the creating images. Err: %s", err)
			handlersfunc.HandleUploadImageError(rw)
		}
		pagesImages = append(pagesImages, localPath)
		log.Println(imageURL)
	}
	
	err = api.ImportImagesFile(pagesImages, pdfName, nil, nil)
	if err != nil {
		log.Printf("Error happened in merging images to pdf. Err: %s", err)
		handlersfunc.HandleUploadImageError(rw)
		return
	}
	err = bucketPdfUpload(pdfName, config.TimewebToken)
	if err != nil {
		log.Printf("Error happened in uploading image to bucket. Err: %s", err)
		handlersfunc.HandleUploadImageError(rw)
		return
	}
	//err = os.Remove(pdfName) 
    //if err != nil { 
    //    log.Printf("Error happened in removing pdf after bucket upload. Err: %s", err)
	//	handlersfunc.HandleUploadImageError(rw)
	//	return
    //} 

	rw.WriteHeader(http.StatusOK)
	rBody.Link = pdfName
	resp["response"] = rBody
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}