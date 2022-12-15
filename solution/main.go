package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"sync"
)

func main() {
	invoiceFilePath := flag.String("invoice", "invoice.txt", "words of an invoice")
	supplierNameFilePath := flag.String("supplier", "suppliernames.txt", "a list of supplier names")
	workerNum := flag.Uint64("worker", 5, "number of workers")
	flag.Parse()

	if err := FindSupplierName(*invoiceFilePath, *supplierNameFilePath, *workerNum); err != nil {
		log.Fatal(err)
	}
}

// FindSupplierName - find the supplier name from input files
// if supplier name is found, it will print the id and supplier name of the supplier
func FindSupplierName(invoiceFilePath, supplierNameFilePath string, workerNum uint64) (err error) {
	if workerNum == 0 {
		return fmt.Errorf("invalid worker num")
	}

	// preprocess the invoice file
	words, err := loadInvoiceFile(invoiceFilePath)
	if err != nil {
		return err
	}
	pages := groupInvoiceWords(words)
	for _, page := range pages {
		sortWordsInPage(page)
	}

	// preprocess the supplier name file
	supplierChan, err := loadSupplierNameFile(supplierNameFilePath)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	done := make(chan bool, 1)
	// send worker job
	for i := uint64(0); i < workerNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runWorker(pages, supplierChan, done)
		}()
	}
	// wait for all worker complete
	wg.Wait()
	select {
	case <-done:
		return nil
	default:
		log.Println("supplier name not found")
	}
	return nil
}

// runWorker - run worker to find the supplier name
func runWorker(pages []*Page, supplierChan chan *Supplier, done chan bool) {
	for supplier := range supplierChan {
		select {
		case <-done: // stop early if other worker has found the supplier name
			done <- true
			return
		default:
			s := SearchSupplierFromPage(pages, supplier)
			if s != nil {
				log.Printf("supplier name found: %s,%s", s.Id, s.SupplierName)
				done <- true
				return
			}
		}
	}
}

// loadSupplierNameFile - load supplier names from file asynchronously
func loadSupplierNameFile(supplierNameFilePath string) (supplierChan chan *Supplier, err error) {
	bufSize := 100
	supplierChan = make(chan *Supplier, bufSize)
	supplierNameFile, err := os.Open(supplierNameFilePath)
	if err != nil {
		return nil, err
	}
	go func() {
		reg := regexp.MustCompile(`(\d+),(.+)`)
		defer supplierNameFile.Close()
		scanner := bufio.NewScanner(supplierNameFile)
		scanner.Scan() // skip the first line
		for scanner.Scan() {
			line := scanner.Text()
			match := reg.FindStringSubmatch(line)
			if len(match) != 3 {
				log.Printf("invalid supplier name text: %s", line)
				return
			}
			id := match[1]
			supplierName := match[2]
			supplierChan <- &Supplier{
				Id:           id,
				SupplierName: supplierName,
			}
		}
		close(supplierChan)
	}()
	return
}

// loadInvoiceFile - load words of an invoice from file
func loadInvoiceFile(invoiceFilePath string) (words []*Word, err error) {
	// use regexp instead of json package because the file content is not valid JSON
	reg := regexp.MustCompile(`'pos_id': (\d+), .+'word': '(.+)', 'line_id': (\d+), .+'page_id': (\d+),`)
	invoiceFile, err := os.Open(invoiceFilePath)
	if err != nil {
		return
	}
	defer invoiceFile.Close()
	words = make([]*Word, 0)
	scanner := bufio.NewScanner(invoiceFile)
	for scanner.Scan() {
		line := scanner.Text()
		match := reg.FindStringSubmatch(line)
		if len(match) != 5 {
			err = fmt.Errorf("invalid invoice text: %s", line)
			return
		}
		posIdStr := match[1]
		word := match[2]
		lineIdStr := match[3]
		pageIdStr := match[4]
		posId, err := strconv.ParseUint(posIdStr, 10, 32)
		if err != nil {
			return nil, err
		}
		lineId, err := strconv.ParseUint(lineIdStr, 10, 32)
		if err != nil {
			return nil, err
		}
		pageId, err := strconv.ParseUint(pageIdStr, 10, 32)
		if err != nil {
			return nil, err
		}
		words = append(words, &Word{
			Word:   word,
			PosId:  uint32(posId),
			LineId: uint32(lineId),
			PageId: uint32(pageId),
		})
	}
	return
}
