package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"strconv"
	"strings"
)

type RedditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				ID      string
				URL     string
				Preview struct {
					Images []struct {
						Source struct {
							Width  int
							Height int
						}
					}
				}
			}
		}
	}
}

const UserAgent = "Reddit Reader v0.13 (by /u/Ptk7l2)"

func main() {
	// grab current user
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	// setup cli flags
	sub := flag.String("subreddit", "EarthPorn", "The subreddit you wish to target for pulling images from")
	sort := flag.String("sort", "new", "Sort for the chosen subreddit")
	limit := flag.String("limit", "100", "Limit for the chosen subreddit")
	width := flag.Int("width", 2560, "Image width minimum for downloading, anything smaller will be rejected")
	height := flag.Int("height", 1440, "Image height minimum for downloading, anything smaller will be rejected")
	dest := flag.String("dest", fmt.Sprintf("%s/Pictures", usr.HomeDir), "The folder path you wisth to save images to")
	flag.Parse()

	// parse url from flags
	url := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%s", *sub, *sort, *limit)

	// Create a request and add the proper headers.
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", UserAgent)

	// Handle the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatal(errors.New(resp.Status))
	}

	// parse reddit response
	var rr RedditResponse
	err = json.NewDecoder(resp.Body).Decode(&rr)
	if err != nil {
		fmt.Println(err.Error())
	}

	// create path if it doesn't exist
	if _, err := os.Stat(*dest); os.IsNotExist(err) {
		os.MkdirAll(*dest, 0700)
	}

	for _, v := range rr.Data.Children {
		imageResponse, e := http.Get(v.Data.URL)
		if e != nil {
			log.Fatal(e)
			continue
		}
		defer imageResponse.Body.Close()

		// create bytes out of image body
		bimage, err := ioutil.ReadAll(imageResponse.Body)
		if err != nil {
			log.Fatal(err)
			continue
		}

		// decode image to check resolution
		bufimagedemension := bytes.NewBuffer(bimage)
		ic, _, err := image.DecodeConfig(bufimagedemension)

		// check image width, if to small will reject
		if ic.Width < *width {
			fmt.Printf("Image width %s to small, skipping...\n", strconv.Itoa(ic.Width))
			continue
		}

		// check image hight, if to small will reject
		if ic.Height < *height {
			fmt.Printf("Image height %s to small, skipping...\n", strconv.Itoa(ic.Height))
			continue
		}

		// set file path
		filePath := fmt.Sprintf("%s/%s.jpg", *dest, strings.Replace(v.Data.ID, " ", "", -1))

		// if path exist move on to next image
		if _, err := os.Stat(filePath); err == nil {
			fmt.Println("Image already downloaded, skipping...")
			continue
		}

		//open a file for writing
		file, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		// Use io.Copy to just dump the response body to the file. This supports huge files
		bufimagereader := bytes.NewBuffer(bimage)
		_, err = io.Copy(file, bufimagereader)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Successfully saved image: %s\n", v.Data.ID)
	}
}
