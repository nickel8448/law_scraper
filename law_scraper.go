package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/cheggaaa/pb/v3"
	"github.com/gocolly/colly"
)

type actDetails struct {
	actID           string
	actNumber       string
	enactmentDate   string
	actYear         string
	shortTitle      string
	longTitle       string
	ministry        string
	department      string
	enforcementDate string
	lastUpdate      string
	location        string
	actURL          string
}

// TODO: Deprecate the allURLs variables
func getAllURLs(startURL string) ([]string, error) {
	fmt.Println("Starting to visit the website")
	c := colly.NewCollector()
	c.AllowURLRevisit = false

	URL := "https://www.indiacode.nic.in"
	urlMap := make(map[string]bool)
	viewURLMap := make(map[string]bool)

	var allURLsQueue []string
	var viewURLs []string
	var URLs []string

	allURLsQueue = append(allURLsQueue, startURL)
	URLs = append(URLs, startURL)

	for len(allURLsQueue) > 0 {
		found := true
		c.OnHTML("a[href]", func(e *colly.HTMLElement) {
			if e.Text == "View..." {
				url := URL + e.Attr("href")
				_, urlInMap := viewURLMap[url]
				if !urlInMap {
					viewURLs = append(viewURLs, url)
					viewURLMap[url] = true
				}
			}
			if e.Attr("class") == "pull-right" {
				url := URL + e.Attr("href")
				_, urlInMap := urlMap[url]
				if !urlInMap {
					allURLsQueue = append(allURLsQueue, url)
					URLs = append(URLs, url)
					// fmt.Println("Adding URL to the queue: ", url)
					urlMap[url] = true
				}
			} else {
				found = false
			}
		})
		if !found {
			break
		}
		c.OnError(func(_ *colly.Response, err error) {
			log.Fatal("Something went wrong: ", err)
			return
		})
		c.Visit(allURLsQueue[0])
		allURLsQueue = allURLsQueue[1:]
	}
	fmt.Println("Loaded all URLs in memory")
	return viewURLs, nil
}

func generateActPDFMap(viewURLs []string) map[actDetails]string {
	fmt.Println("Starting to load Acts data in memory")
	counter := 0
	count := len(viewURLs)
	bar := pb.StartNew(count)
	URL := "https://www.indiacode.nic.in"
	c := colly.NewCollector()
	c.AllowURLRevisit = false

	actPDFMap := make(map[actDetails]string)

	pdfSet := make(map[string]bool)

	for _, url := range viewURLs {
		var pdfURL string
		var currentAct actDetails
		c.OnHTML("body", func(e *colly.HTMLElement) {
			goquerySelection := e.DOM
			goquerySelection.Find("a[href]").Each(func(_ int, u *goquery.Selection) {
				attr, _ := u.Attr("href")
				if strings.Contains(attr, "/bitstream") {
					pdfURLTemp := URL + attr
					_, pdfInSet := pdfSet[pdfURLTemp]
					if !pdfInSet {
						pdfSet[pdfURL] = true
						pdfURL = pdfURLTemp
					}
				}
			})
			goquerySelection.Find(".table.itemDisplayTable").Each(func(_ int, table *goquery.Selection) {
				table.Find("tr").Each(func(_ int, s *goquery.Selection) {
					s.Find("td").Each(func(_ int, d *goquery.Selection) {
						if strings.Contains(d.Text(), "Act ID") {
							currentAct.actID = d.Next().Text()
						} else if strings.Contains(d.Text(), "Act Num") {
							currentAct.actNumber = d.Next().Text()
						} else if strings.Contains(d.Text(), "Enactment Date") {
							currentAct.enactmentDate = d.Next().Text()
						} else if strings.Contains(d.Text(), "Act Year") {
							currentAct.actYear = d.Next().Text()
						} else if strings.Contains(d.Text(), "Short Title") {
							currentAct.shortTitle = d.Next().Text()
						} else if strings.Contains(d.Text(), "Long Title") {
							currentAct.longTitle = d.Next().Text()
						} else if strings.Contains(d.Text(), "Ministry") {
							currentAct.ministry = d.Next().Text()
						} else if strings.Contains(d.Text(), "Department") {
							currentAct.department = d.Next().Text()
						} else if strings.Contains(d.Text(), "Enforcement Date") {
							currentAct.enforcementDate = d.Next().Text()
						} else if strings.Contains(d.Text(), "Last Updated") {
							currentAct.lastUpdate = d.Next().Text()
						} else if strings.Contains(d.Text(), "Location") {
							currentAct.lastUpdate = d.Next().Text()
						}
					})
				})
			})
		})
		currentAct.actURL = url
		c.Visit(url)
		bar.Increment()
		actPDFMap[currentAct] = pdfURL
		if counter == 50 {
			break
		}
		counter++
	}
	bar.Finish()
	fmt.Println("Acts data loaded in memory")
	return actPDFMap
}

func downloadPDFAndAddDataToCSV(actPDFMap map[actDetails]string) {
	fmt.Println("Downloading PDFs and adding data to CSV")
	count := len(actPDFMap)
	bar := pb.StartNew(count)
	c := colly.NewCollector()
	c.AllowURLRevisit = false
	os.Mkdir("index", os.ModePerm)
	os.Mkdir("pdfs", os.ModePerm)
	file, err := os.Create("index/data.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	csvHeader := []string{"Act ID",
		"Act Num",
		"Enactment Date",
		"Act Year",
		"Short Title",
		"Long Title",
		"Ministry",
		"Department",
		"Enforcement Date",
		"Last Updated",
		"Location",
		"Act URL"}
	writer.Write(csvHeader)
	defer writer.Flush()

	for actDetails, pdfURL := range actPDFMap {
		var rowSlice []string
		rowSlice = append(rowSlice,
			actDetails.actID,
			actDetails.actNumber,
			actDetails.enactmentDate,
			actDetails.actYear,
			actDetails.shortTitle,
			actDetails.longTitle,
			actDetails.ministry,
			actDetails.department,
			actDetails.enforcementDate,
			actDetails.lastUpdate,
			actDetails.location,
			actDetails.actURL)

		c.OnResponse(func(r *colly.Response) {
			// fileName := r.FileName()
			err := r.Save("pdfs/" + actDetails.actID + "_" + actDetails.actNumber + ".pdf")
			if err != nil {
				log.Fatal(err)
			}
		})
		c.Visit(pdfURL)
		rowSlice = append(rowSlice, "pdfs/"+actDetails.actID+"_"+actDetails.actNumber+".pdf")
		writer.Write(rowSlice)
		bar.Increment()
	}
	bar.Finish()
	fmt.Println("Done")
}

func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 2 || len(argsWithProg) >= 3 {
		log.Fatal("Arguments are not right. Please add the URL")
	}
	viewURLs, err := getAllURLs(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	actPDFMap := generateActPDFMap(viewURLs)
	downloadPDFAndAddDataToCSV(actPDFMap)
}
