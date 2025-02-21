// Handlers package contains endpoints handlers for the Photo Book Editor module.
//
// https://github.com/SiberianMonster/memoryprint/tree/development/internal/imagehandlers
package imagehandlers

import (
	"encoding/json"
	"context"
	"log"
	"net/http"
	"image"
	"image/png"
	"image/jpeg"
	"image/draw"
    "os"
    "bytes"
	"strings"
	"strconv"
	"io/ioutil"
	"errors"
	"github.com/gorilla/mux"
	"sort"

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
	"github.com/SiberianMonster/memoryprint/internal/userstorage"
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

	
type FolderContent struct {
	Meta struct {
		Total int `json:"total"`
	} `json:"meta"`
	Files []any `json:"files"`
}

type NewDirectory struct {
    DirName    string `json:"dir_name"`
}

type CopyPasteImage struct {
    Destination string   `json:"destination"`
	Source      []string `json:"source"`
}

type RenameImage struct {
    NewFilename string   `json:"new_filename"`
	OldFilename      string `json:"old_filename"`
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
	} else if strings.Contains(filename, "jpg") {
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

	//form := new(bytes.Buffer)
	//writer := multipart.NewWriter(form)
	//fw, err := writer.CreateFormFile(filename, filepath.Base(filename))
	//if err != nil {
	//	log.Printf("Failed to create form file %s", err)
	//	return err
	//}
	fd, err := os.Open(filename)
	if err != nil {
		log.Printf("Failed to open file %s", err)
		return err
	}
	defer fd.Close()
	//_, err = io.Copy(fw, fd)
	//if err != nil {
	//	log.Printf("Failed to copy file content %s", err)
	//	return err
	//}

	//writer.Close()
	tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }

	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("POST", "https://api.timeweb.cloud/api/v1/storages/buckets/225285/object-manager/upload?;path=photo/", fd)
	if err != nil {
		log.Printf("Failed to create a request to bucket %s", err)
	}
	req.Header.Set("Authorization", "Bearer " + timewebToken)
	//req.Header.Set("Content-Type", writer.FormDataContentType())
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
			name: filename,
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
	fw, err := writer.CreateFormFile("mediaFile", filename)
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
			rw.WriteHeader(http.StatusOK)
			resp := make(map[string]handlersfunc.ValidationErrorBody)
			var errorB handlersfunc.ValidationErrorBody
			errorB.ErrorCode = 422
			errorB.ErrorMessage = "Validation failed"
			out := map[string][]string{} 
			errorB.Errors = out
			resp["error"] = errorB
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				log.Printf("Error happened in JSON marshal. Err: %s", err)
				return
			}
			rw.Write(jsonResp)
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
		filename = "./temp_photo/"+code+"_img.png"

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

	userCheck := userstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw)
		return
	}
	
	var leatherID *uint
	pages, err := projectstorage.RetrieveProjectPages(ctx, config.DB, projectID, false, leatherID, "TEMPLATE")
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

		strCreatingImageLink := *page.PreviewImageLink
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

func CreatePrintVersion(pID uint, images []models.ExportPage, variant string) error {


	folderName := "photobook_" + strconv.Itoa(int(pID))
	// Sort the slice by age in descending order
    sort.Slice(images, func(i, j int) bool {
        return images[i].Sort < images[j].Sort
    })
	
	
	for i, page := range images {
		var strCreatingImageLink string
		var imageURL string
		var localPath string
		var previousI int
		var previousPath string
		var spinePath string
		var midPath string
		var frontPath string
		var localFolder string

		strCreatingImageLink = page.PreviewImageLink
		imageURL = config.ImageHost+strCreatingImageLink
		localPath = "/" + folderName + "/" + strconv.Itoa(int(page.Sort)) + ".png"
		log.Println(localPath)
		localFolder = "/" + folderName

		if err = os.MkdirAll(localFolder, os.ModePerm); err != nil {
			log.Printf("Error happened in creating folder. Err: %s", err)
			return err
		}
		err = DownloadFile(localPath, imageURL) 
		if err != nil {
			log.Printf("Error happened in loading the creating images. Err: %s", err)
			return err
		}
		log.Println(imageURL)
		if i%2 == 0 && i != 0 && i < len(images) - 1 && variant == "PREMIUM" {
			previousI = i - 1
			previousPath = "/" + folderName + "/" + strconv.Itoa(previousI) + ".png"
			err = MergeImages(previousPath, localPath)
			if err != nil {
				log.Printf("Error happened in merging the images. Err: %s", err)
				return err
			}
		}
		if i == len(images) - 1 {
			spinePath = "/" + folderName + "/" + strconv.Itoa(1000) + ".png"
			err = MergeImages(spinePath, localPath)
			if err != nil {
				log.Printf("Error happened in merging the images. Err: %s", err)
				return err
			}
			stringSlice := strings.Split(localPath, ".")
			midPath = stringSlice[0] + "_appended.png"
			frontPath = "/" + folderName + "/" + strconv.Itoa(0) + ".png"
			err = MergeImages(frontPath, midPath)
			if err != nil {
				log.Printf("Error happened in merging the images. Err: %s", err)
				return err
			}

		} 

	}
	return nil

}

func CreateProjectFolder(images []models.ExportPage, pID uint) {

	folderName := "photobook_" + strconv.Itoa(int(pID))
	var folderContent FolderContent

	//check if folder exists
	urlCheck := "https://api.timeweb.cloud/api/v1/storages/buckets/225285/object-manager/list?prefix=" + folderName + "/"
	client := &http.Client{}
	req, err := http.NewRequest("GET", urlCheck, nil)
	if err != nil {
		log.Printf("Failed to create a request to bucket %s", err)
		log.Println(pID)
	}
	req.Header.Set("Authorization", "Bearer " + config.TimewebToken)
	req.Header.Set("Content-Type","application/json")
	resp, err := client.Do(req)
	if err != nil {
			log.Printf("Failed to make a request to bucket %s", err)
			log.Println(pID)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
            log.Println(resp.StatusCode)
			log.Printf("Failed to make a request to bucket %s", err)
			log.Println(pID)
    } else {
		err = json.NewDecoder(resp.Body).Decode(&folderContent)
		log.Println(folderContent)
		if err != nil {
			log.Printf("Failed to decode folder content %s", err)
			log.Println(pID)
		}
		if folderContent.Meta.Total == len(images){
				log.Println("all pages already copied")
				log.Println(pID)
		} else {  //copy paste images
			log.Println("missing pages, start to copy")
			log.Println(pID)

			//create folder
			body := &NewDirectory{
				DirName:    folderName,
			}
			
			urlCreateFolder := "https://api.timeweb.cloud/api/v1/storages/buckets/225285/object-manager/mkdir"
			payloadBuf := new(bytes.Buffer)
			json.NewEncoder(payloadBuf).Encode(body)
			req, err = http.NewRequest("POST", urlCreateFolder, payloadBuf)
			if err != nil {
				log.Printf("Failed to create a request to create new folder %s", err)
				log.Println(pID)
			}
			req.Header.Set("Authorization", "Bearer " + config.TimewebToken)
			req.Header.Set("Content-Type","application/json")
			resp, err = client.Do(req)
			if err != nil {
					log.Printf("Failed to make a request to create new folder %s", err)
					log.Println(pID)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 201 {
					log.Println(resp.StatusCode)
					log.Printf("Failed to make a request to create new folder %s", err)
					log.Println(pID)
			} else {
				var pathToCopy string
				urlCopyPasteImage := "https://api.timeweb.cloud/api/v1/storages/buckets/225285/object-manager/copy"
				for _, image := range images {
					// copy each page
					body := &CopyPasteImage{
						Destination:    folderName,
					}
					pathToCopy = "photo/" + image.PreviewImageLink
					body.Source = append(body.Source, pathToCopy)

					payloadBuf = new(bytes.Buffer)
					json.NewEncoder(payloadBuf).Encode(body)
					req, err = http.NewRequest("POST", urlCopyPasteImage, payloadBuf)
					if err != nil {
						log.Printf("Failed to create a request to copy page %s", err)
						log.Println(pID)
					}
					req.Header.Set("Authorization", "Bearer " + config.TimewebToken)
					req.Header.Set("Content-Type","application/json")
					resp, err = client.Do(req)
					if err != nil {
							log.Printf("Failed to make a request to copy page %s", err)
							log.Println(pID)
					}
					defer resp.Body.Close()
					if resp.StatusCode != 204 {
							log.Println(resp.StatusCode)
							log.Printf("Failed to make a request to copy page %s", err)
							log.Println(pID)
							break
					}
					oldName := folderName + "/" + image.PreviewImageLink
					newName := folderName + "/" + strconv.Itoa(int(image.Sort)) + ".png"
					renameData := &RenameImage{
						NewFilename: newName,
						OldFilename: oldName,
					}
					payloadBuf = new(bytes.Buffer)
					json.NewEncoder(payloadBuf).Encode(renameData)
					
					req, err := http.NewRequest("POST", "https://api.timeweb.cloud/api/v1/storages/buckets/1051/object-manager/rename", payloadBuf)
					if err != nil {
						log.Printf("Failed to createb request to rename page %s", err)
						log.Println(pID)
					}
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer " + config.TimewebToken)
					resp, err := client.Do(req)
					if err != nil {
						log.Printf("Failed to rename page %s", err)
						log.Println(pID)
					}
					defer resp.Body.Close()
					}

				log.Println("finished copying pages")
				log.Println(pID)

				}
			}
		}
	
	
}

func CheckProjectFolder(pID uint) bool {

	folderName := "photobook_" + strconv.Itoa(int(pID))
	var folderContent FolderContent

	//check if folder exists
	urlCheck := "https://api.timeweb.cloud/api/v1/storages/buckets/225285/object-manager/list?prefix=" + folderName + "/"
	client := &http.Client{}
	req, err := http.NewRequest("GET", urlCheck, nil)
	if err != nil {
		log.Printf("Failed to create a request to bucket %s", err)
		log.Println(pID)
	}
	req.Header.Set("Authorization", "Bearer " + config.TimewebToken)
	req.Header.Set("Content-Type","application/json")
	resp, err := client.Do(req)
	if err != nil {
			log.Printf("Failed to make a request to bucket %s", err)
			log.Println(pID)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
            log.Println(resp.StatusCode)
			log.Printf("Failed to make a request to bucket %s", err)
			log.Println(pID)
    } else {
		err = json.NewDecoder(resp.Body).Decode(&folderContent)
		if err != nil {
			log.Printf("Failed to decode folder content %s", err)
			log.Println(pID)
		}
		if folderContent.Meta.Total > 0 {
			log.Println("pages already copied")
			log.Println(pID)
			return true
		}
	}
	return false
} 

func MergeImages(firstImage string, secondImage string) error {

	imgFile1, err := os.Open(firstImage)
	if err != nil {
		log.Printf("Failed to open image for merging %s", err)
		return err
	}
	imgFile2, err := os.Open(secondImage)
	if err != nil {
		log.Printf("Failed to open image for merging %s", err)
		return err
	}
	img1, _, err := image.Decode(imgFile1)
	if err != nil {
		log.Printf("Failed to decode image for merging %s", err)
		return err
	}
	img2, _, err := image.Decode(imgFile2)
	if err != nil {
		log.Printf("Failed to decode image for merging %s", err)
		return err
	}

	sp2 := image.Point{img1.Bounds().Dx(), 0}
	r2 := image.Rectangle{sp2, sp2.Add(img2.Bounds().Size())}
	r := image.Rectangle{image.Point{0, 0}, r2.Max}
	rgba := image.NewRGBA(r)
	draw.Draw(rgba, img1.Bounds(), img1, image.Point{0, 0}, draw.Src)
	draw.Draw(rgba, r2, img2, image.Point{0, 0}, draw.Src)
	stringSlice := strings.Split(secondImage, ".")
	new_name := stringSlice[0] + "_appended.png"
	out, err := os.Create(new_name)
	if err != nil {
		log.Printf("Failed to export merged image  %s", err)
		return err
	}

	png.Encode(out, rgba)
	return nil
}